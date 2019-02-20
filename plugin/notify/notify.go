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
			ObjectNeedParams: []int{event.ObjectNeedCheckEvent},
			Handler:          p.handleCheckSuiteEvent,
		},
		plugin.HandlerOptions{
			Events:           []string{event.EvCheckRun},
			Actions:          []string{event.ActionCompleted},
			ObjectNeedParams: []int{event.ObjectNeedCheckEvent},
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

func (p *NotifyPlugin) handleCheckSuiteEvent(ctx *event.EventContext) (err error) {
	if p.extra.CheckSuiteComplete == nil {
		return
	}

	checkEvent, _ := ctx.Object.CheckEvent()
	suite := checkEvent.Suite

	log.Debug("check suite: %+v", *suite)
	prs, err := p.cli.ListPullRequestBySHA(ctx.Ctx, ctx.Owner, ctx.Repo, suite.HeadSHA)
	if err != nil {
		return fmt.Errorf("list pull request by sha error: %v", err)
	}

	log.Debug("pull requests: %v", prs)
	if len(prs) > 0 {
		pr := prs[0]
		notifyOption, ok := p.extra.CheckSuiteComplete.Authors[pr.User]
		if !ok {
			notifyOption = p.extra.CheckSuiteComplete.Default
		}

		content := fmt.Sprintf("check suite complete, status [%s], conclusion [%s]\n", suite.Status, suite.Conclusion)
		content += fmt.Sprintf("Title [%s] Author [%s]\n%s", pr.Title, pr.User, pr.HTMLURL)
		log.Debug("check suite [%s] [%s] [%s], send notify", pr.Title, suite.Status, suite.Conclusion)
		err = p.notifier.Send(ctx.Ctx, &notifyOption, content)
		return err
	}
	return
}

func (p *NotifyPlugin) handleCheckRunEvent(ctx *event.EventContext) (err error) {
	if p.extra.CheckRunComplete == nil {
		return
	}

	checkEvent, _ := ctx.Object.CheckEvent()
	run := checkEvent.Run

	log.Debug("check run: %+v", *run)
	prs, err := p.cli.ListPullRequestBySHA(ctx.Ctx, ctx.Owner, ctx.Repo, run.HeadSHA)
	if err != nil {
		return fmt.Errorf("list pull request by sha error: %v", err)
	}

	log.Debug("pull requests: %v", prs)
	if len(prs) > 0 {
		pr := prs[0]
		notifyOption, ok := p.extra.CheckRunComplete.Authors[pr.User]
		if !ok {
			notifyOption = p.extra.CheckRunComplete.Default
		}

		content := fmt.Sprintf("check run complete, status [%s], conclusion [%s]\n", run.Status, run.Conclusion)
		content += fmt.Sprintf("Title [%s] Author [%s]\n%s", pr.Title, pr.User, pr.HTMLURL)
		log.Debug("check run [%s] [%s] [%s], send notify", pr.Title, run.Status, run.Conclusion)
		err = p.notifier.Send(ctx.Ctx, &notifyOption, content)
		return err
	}
	return
}
