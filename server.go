package freebot

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/httputil"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/plugin"
	_ "github.com/fatedier/freebot/plugin/assign"
	_ "github.com/fatedier/freebot/plugin/merge"
	_ "github.com/fatedier/freebot/plugin/status"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Config struct {
	BindAddr          string `json:"bind_addr"`
	LogLevel          string `json:"log_level"`
	LogFile           string `json:"log_file"`
	LogMaxDays        int64  `json:"log_max_days"`
	GithubAccessToken string `json:"github_access_token"`

	// owner -> repo -> plugin
	RepoConfs map[string]map[string]RepoConf `json:"repo_confs"`
}

type RepoConf struct {
	Alias   config.AliasOptions     `json:"alias"`
	Roles   config.RoleOptions      `json:"roles"`
	Plugins map[string]PluginConfig `json:"plugins"`
}

type PluginConfig struct {
	Enable        bool                  `json:"enable"`
	Preconditions []config.Precondition `json:"preconditions"`
	Extra         interface{}           `json:"extra"`
}

type Service struct {
	Config

	eventHandler *EventHandler
}

func NewService(cfg Config) (*Service, error) {
	if cfg.LogMaxDays <= 0 {
		cfg.LogMaxDays = 3
	}
	if cfg.LogFile == "" {
		log.InitLog("console", "", cfg.LogLevel, cfg.LogMaxDays)
	} else {
		log.InitLog("file", cfg.LogFile, cfg.LogLevel, cfg.LogMaxDays)
	}

	svc := &Service{
		Config: cfg,
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GithubAccessToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	githubCli := github.NewClient(tc)
	cli := client.NewGithubClient(githubCli)

	plugins := make(map[string][]plugin.Plugin)
	for owner, repos := range cfg.RepoConfs {
		for repo, repoConf := range repos {
			log.Info("repo [%s/%s] alias: %+v", owner, repo, repoConf.Alias)
			log.Info("repo [%s/%s] roles: %+v", owner, repo, repoConf.Roles)
			for pluginName, pluginConf := range repoConf.Plugins {
				if !pluginConf.Enable {
					continue
				}

				baseOptions := plugin.PluginOptions{}
				baseOptions.Complete(owner, repo, repoConf.Alias, repoConf.Roles, pluginConf.Preconditions, pluginConf.Extra)
				p, err := plugin.Create(cli, pluginName, baseOptions)
				if err != nil {
					err = fmt.Errorf("create plugin [%s] error: %v", pluginName, err)
					log.Error("%v", err)
					return nil, err
				}

				arrs, ok := plugins[owner+"/"+repo]
				if ok {
					arrs = append(arrs, p)
				} else {
					arrs = make([]plugin.Plugin, 1)
					arrs[0] = p
				}
				plugins[owner+"/"+repo] = arrs
			}
		}
	}

	svc.eventHandler = NewEventHandler(plugins)
	return svc, nil
}

func (svc *Service) Run() error {
	log.Info("freebot listen on %s", svc.BindAddr)
	err := http.ListenAndServe(svc.BindAddr, http.HandlerFunc(svc.Handler))
	return err
}

func (svc *Service) Handler(w http.ResponseWriter, r *http.Request) {
	eventType := r.Header.Get("X-Github-Event")
	if eventType == "" {
		httputil.ReplyError(w, httputil.NewHttpError(400, "unsupport event"))
		return
	}

	log.Debug("event [%s], id [%s]", eventType, r.Header.Get("X-GitHub-Delivery"))

	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warn("read request body error: %v", err)
		httputil.ReplyError(w, httputil.NewHttpError(400, "read request error"))
		return
	}

	err = svc.eventHandler.HandleEvent(r.Context(), eventType, string(content))
	if err != nil {
		log.Warn("handle event error: %v", err)
		httputil.ReplyError(w, err)
		return
	}

	w.WriteHeader(200)
}
