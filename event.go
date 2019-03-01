package freebot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/client/githubapp"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/httputil"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/plugin"

	"github.com/google/go-github/github"
)

var (
	ErrEventPayload   = httputil.NewHttpError(400, "error event payload")
	ErrNoSupportEvent = httputil.NewHttpError(400, "no support event")
	ErrNoOwnerRepo    = httputil.NewHttpError(400, "event no owner and repo info")
	ErrNoPlugins      = httputil.NewHttpError(400, "no correspond plugins")
	ErrNoInstallation = httputil.NewHttpError(400, "no installation")
)

type EventHandler struct {
	requireInstallation bool

	// key is owner/repo
	plugins map[string][]plugin.Plugin

	mu sync.RWMutex
}

func NewEventHandler(requireInstallation bool, plugins map[string][]plugin.Plugin) *EventHandler {
	return &EventHandler{
		requireInstallation: requireInstallation,
		plugins:             plugins,
	}
}

func (eh *EventHandler) UpdatePlugins(plugins map[string][]plugin.Plugin) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	// close all plugin
	for _, ps := range eh.plugins {
		for _, p := range ps {
			pv, valid := p.(plugin.TaskRunner)
			if valid {
				pv.Close()
			}
		}
	}

	eh.plugins = plugins
}

func (eh *EventHandler) HandleEvent(ctx context.Context, evType string, content string) (err error) {
	var (
		payload interface{}
		owner   string
		repo    string
	)

	// parse content
	switch evType {
	case event.EvIssueComment:
		v := &github.IssueCommentEvent{}
		err = json.Unmarshal([]byte(content), &v)
		payload = v
	case event.EvPullRequest:
		v := &github.PullRequestEvent{}
		err = json.Unmarshal([]byte(content), &v)
		payload = v
	case event.EvPullRequestReview:
		v := &github.PullRequestReviewEvent{}
		err = json.Unmarshal([]byte(content), &v)
		payload = v
	case event.EvPullRequestReviewComment:
		v := &github.PullRequestReviewCommentEvent{}
		err = json.Unmarshal([]byte(content), &v)
		payload = v
	case event.EvCheckRun:
		v := &github.CheckRunEvent{}
		err = json.Unmarshal([]byte(content), &v)
		payload = v
	case event.EvCheckSuite:
		v := &github.CheckSuiteEvent{}
		err = json.Unmarshal([]byte(content), &v)
		payload = v
	case event.EvPing:
		return nil
	default:
		return ErrNoSupportEvent
	}

	if err != nil {
		return ErrEventPayload
	}

	// get owner and repo name
	if v, ok := payload.(client.GetRepoInterface); ok {
		owner = v.GetRepo().GetOwner().GetLogin()
		repo = v.GetRepo().GetName()
	} else {
		return ErrNoOwnerRepo
	}

	if eh.requireInstallation {
		if v, ok := payload.(client.GetInstallationInterface); ok && v.GetInstallation() != nil && v.GetInstallation().ID != nil {
			ctx = githubapp.WithInstallID(ctx, int(*v.GetInstallation().ID))
		} else {
			return ErrNoInstallation
		}
	}

	// get plugins
	eh.mu.RLock()
	plugins, ok := eh.plugins[owner+"/"+repo]
	eh.mu.RUnlock()
	if !ok {
		return ErrNoPlugins
	}

	// handle event by plugins
	var (
		notSupport bool
		partialErr error
	)
	object := client.NewObject(payload)
	for _, p := range plugins {
		notSupport, partialErr = p.HandleEvent(&event.EventContext{
			Ctx:    ctx,
			Type:   evType,
			Owner:  owner,
			Repo:   repo,
			Object: object,
		})
		if notSupport {
			log.Debug("[%s/%s] plugin [%s] not support", owner, repo, p.Name())
			continue
		}

		log.Info("[%s/%s] plugin: [%s] event: [%v]", owner, repo, p.Name(), evType)
		if partialErr != nil {
			err = fmt.Errorf("%v;[%s] %v", err, p.Name(), partialErr)
		}
	}
	return err
}
