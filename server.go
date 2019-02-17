package freebot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

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

	// repo -> plugin
	RepoConfs map[string]RepoConf `json:"repo_confs"`

	RepoConfDir                string `json:"repo_conf_dir"`
	RepoConfDirUpdateIntervalS int    `json:"repo_conf_dir_update_interval_s"`
}

type RepoConf struct {
	Alias   config.AliasOptions     `json:"alias"`
	Roles   config.RoleOptions      `json:"roles"` // role -> []string{user1, user2}
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
	cli          client.ClientInterface

	staticRepoConfs map[string]RepoConf
	extraRepoConfs  map[string]RepoConf
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
	if cfg.RepoConfDirUpdateIntervalS <= 0 {
		cfg.RepoConfDirUpdateIntervalS = 5
	}

	svc := &Service{
		Config: cfg,
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GithubAccessToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	githubCli := github.NewClient(tc)
	svc.cli = client.NewGithubClient(githubCli)

	svc.staticRepoConfs = cfg.RepoConfs
	if svc.RepoConfDir != "" {
		extraRepoConfs, err := svc.loadRepoConfsFromDir(svc.RepoConfDir)
		if err != nil {
			return nil, fmt.Errorf("load repo confs from dir error: %v", err)
		}

		svc.extraRepoConfs = extraRepoConfs
	}
	repoConfs := svc.mergeRepoConfsTo(nil, svc.staticRepoConfs)
	repoConfs = svc.mergeRepoConfsTo(repoConfs, svc.extraRepoConfs)
	plugins, err := svc.createPlugins(repoConfs)
	if err != nil {
		return nil, fmt.Errorf("create plugins error: %v", err)
	}

	svc.eventHandler = NewEventHandler(plugins)
	return svc, nil
}

func (svc *Service) Run() error {
	go svc.updatePluginsWorker()

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

func (svc *Service) loadRepoConfsFromDir(path string) (map[string]RepoConf, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	out := make(map[string]RepoConf)
	for _, file := range files {
		if !file.IsDir() {
			fpath := filepath.Join(path, file.Name())
			buf, err := ioutil.ReadFile(fpath)
			if err != nil {
				return nil, err
			}

			tmp := make(map[string]RepoConf)
			err = json.Unmarshal(buf, &tmp)
			if err != nil {
				return nil, fmt.Errorf("parse file [%s] error: %v", fpath, err)
			}
			out = svc.mergeRepoConfsTo(out, tmp)
		}
	}
	return out, nil
}

func (svc *Service) mergeRepoConfsTo(dst map[string]RepoConf, src map[string]RepoConf) map[string]RepoConf {
	if dst == nil {
		dst = make(map[string]RepoConf)
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (svc *Service) createPlugins(repoConfs map[string]RepoConf) (plugins map[string][]plugin.Plugin, err error) {
	plugins = make(map[string][]plugin.Plugin)
	for repoName, repoConf := range repoConfs {
		log.Info("repo [%s] alias: %+v", repoName, repoConf.Alias)
		log.Info("repo [%s] roles: %+v", repoName, repoConf.Roles)

		for pluginName, pluginConf := range repoConf.Plugins {
			if !pluginConf.Enable {
				continue
			}

			arrs := strings.Split(repoName, "/")
			if len(arrs) < 2 {
				return nil, fmt.Errorf("repo name invalid")
			}
			baseOptions := plugin.PluginOptions{}
			baseOptions.Complete(arrs[0], arrs[1], repoConf.Alias, repoConf.Roles, pluginConf.Preconditions, pluginConf.Extra)
			p, err := plugin.Create(svc.cli, pluginName, baseOptions)
			if err != nil {
				err = fmt.Errorf("create plugin [%s] error: %v", pluginName, err)
				log.Error("%v", err)
				return nil, err
			}

			ps, ok := plugins[repoName]
			if ok {
				ps = append(ps, p)
			} else {
				ps = make([]plugin.Plugin, 1)
				ps[0] = p
			}
			plugins[repoName] = ps
		}
	}
	return plugins, nil
}

func (svc *Service) updatePluginsWorker() {
	for {
		time.Sleep(time.Duration(svc.RepoConfDirUpdateIntervalS) * time.Second)
		if svc.RepoConfDir != "" {
			repoConfs, err := svc.loadRepoConfsFromDir(svc.RepoConfDir)
			if err != nil {
				log.Error("load repo confs from dir error: %v", err)
				continue
			}

			if !reflect.DeepEqual(svc.extraRepoConfs, repoConfs) {
				log.Info("repo confs changed...")
				all := svc.mergeRepoConfsTo(nil, repoConfs)
				svc.mergeRepoConfsTo(all, svc.staticRepoConfs)
				plugins, err := svc.createPlugins(all)
				if err != nil {
					log.Error("create plugins error: %v", err)
					continue
				}

				svc.eventHandler.UpdatePlugins(plugins)
				log.Info("update plugins success")

				svc.extraRepoConfs = repoConfs
			}
		}
	}
}
