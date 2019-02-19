package notify

import (
	"fmt"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/notify"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName = "notify"
)

func init() {
	plugin.Register(PluginName, NewNotifyPlugin)
}

type Extra struct {
	CheckSuiteComplete *AuthorNotify `json:"check_suite_complete,omitempty"`
	CheckRunComplete   *AuthorNotify `json:"check_run_complete,omitempty"`
}

type AuthorNotify struct {
	Default notify.NotifyOptions            `json:"default"`
	Authors map[string]notify.NotifyOptions `json:"authors"`
}

type NotifyPlugin struct {
	*plugin.BasePlugin

	extra    Extra
	cli      client.ClientInterface
	notifier notify.NotifyInterface
}

func NewNotifyPlugin(cli client.ClientInterface, notifier notify.NotifyInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &NotifyPlugin{
		cli:      cli,
		notifier: notifier,
	}
	handlerOptions := []plugin.HandlerOptions{
		plugin.HandlerOptions{
			Events:           []string{event.EvCheckSuite},
			Actions:          []string{event.ActionCompleted},
			ObjectNeedParams: []int{event.ObjectNeedCheckSuiteStatus, event.ObjectNeedCheckSuiteConclusion},
			Handler:          p.hanldeCheckSuiteEvent,
		},
		plugin.HandlerOptions{
			Events:           []string{event.EvCheckRun},
			Actions:          []string{event.ActionCompleted},
			ObjectNeedParams: []int{event.ObjectNeedCheckRunStatus, event.ObjectNeedCheckRunConclusion},
			Handler:          p.handleCheckRunEvent,
		},
	}
	options.Handlers = handlerOptions

	p.BasePlugin = plugin.NewBasePlugin(PluginName, options)

	err := p.UnmarshalTo(&p.extra)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *NotifyPlugin) hanldeCheckSuiteEvent(ctx *event.EventContext) (err error) {
	if p.extra.CheckSuiteComplete == nil {
		return
	}

	status, _ := ctx.Object.CheckSuiteStatus()
	conclusion, _ := ctx.Object.CheckSuiteConclusion()

	notifyOptions := &p.extra.CheckSuiteComplete.Default
	content := fmt.Sprintf("one check suite complete, status [%s], conclusion [%s]", status, conclusion)
	log.Debug("check suite [%s] [%s], send notify", status, conclusion)
	err = p.notifier.Send(ctx.Ctx, notifyOptions, content)
	return err
}

func (p *NotifyPlugin) handleCheckRunEvent(ctx *event.EventContext) (err error) {
	if p.extra.CheckRunComplete == nil {
		return
	}

	status, _ := ctx.Object.CheckRunStatus()
	conclusion, _ := ctx.Object.CheckRunConclusion()

	notifyOptions := &p.extra.CheckRunComplete.Default
	content := fmt.Sprintf("one check run complete, status [%s], conclusion [%s]", status, conclusion)
	log.Debug("check run [%s] [%s], send notify", status, conclusion)
	err = p.notifier.Send(ctx.Ctx, notifyOptions, content)
	return err
}
