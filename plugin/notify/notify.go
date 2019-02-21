package notify

import (
	"fmt"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/notify"
	"github.com/fatedier/freebot/plugin"
)

const (
	NotifyCheckRunComplete   = "check_run_complete"
	NotifyCheckSuiteComplete = "check_suite_complete"
)

var (
	PluginName = "notify"
)

func init() {
	plugin.Register(PluginName, NewNotifyPlugin)
}

type Extra struct {
	UserNotifyConfs map[string]*notify.NotifyOptions `json:"user_notify_confs"`
	Events          map[string]*EventNotifyConf      `json:"events"`
}

func (ex *Extra) Complete() {
	if ex.UserNotifyConfs == nil {
		ex.UserNotifyConfs = make(map[string]*notify.NotifyOptions)
	}
	if ex.Events == nil {
		ex.Events = make(map[string]*EventNotifyConf)
	}
	for _, notifyConf := range ex.Events {
		if notifyConf != nil {
			notifyConf.UsersMap = make(map[string]struct{})
			for _, user := range notifyConf.Users {
				notifyConf.UsersMap[user] = struct{}{}
			}
		}
	}
}

type EventNotifyConf struct {
	DefaultUser string              `json:"default_user"`
	Users       []string            `json:"users"`
	UsersMap    map[string]struct{} `json:"-"`
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
	p.extra.Complete()
	return p, nil
}

func (p *NotifyPlugin) handleCheckSuiteEvent(ctx *event.EventContext) (err error) {
	conf, ok := p.extra.Events[NotifyCheckSuiteComplete]
	if !ok {
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
		notifyOption, err := p.getNotifyOption(NotifyCheckSuiteComplete, pr.User, conf)
		if err != nil {
			log.Warn("%v", err)
			return err
		}

		content := fmt.Sprintf("check suite complete, status [%s], conclusion [%s]\n", suite.Status, suite.Conclusion)
		content += fmt.Sprintf("Title [%s] Author [%s]\n%s", pr.Title, pr.User, pr.HTMLURL)
		log.Debug("check suite [%s] [%s] [%s], send notify", pr.Title, suite.Status, suite.Conclusion)
		err = p.notifier.Send(ctx.Ctx, notifyOption, content)
		return err
	}
	return
}

func (p *NotifyPlugin) handleCheckRunEvent(ctx *event.EventContext) (err error) {
	conf, ok := p.extra.Events[NotifyCheckRunComplete]
	if !ok {
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
		notifyOption, err := p.getNotifyOption(NotifyCheckRunComplete, pr.User, conf)
		if err != nil {
			log.Warn("%v", err)
			return err
		}

		content := fmt.Sprintf("check run complete, status [%s], conclusion [%s]\n", run.Status, run.Conclusion)
		content += fmt.Sprintf("Title [%s] Author [%s]\n%s", pr.Title, pr.User, pr.HTMLURL)
		log.Debug("check run [%s] [%s] [%s], send notify", pr.Title, run.Status, run.Conclusion)
		err = p.notifier.Send(ctx.Ctx, notifyOption, content)
		return err
	}
	return
}

func (p *NotifyPlugin) getNotifyOption(eventName string, user string, conf *EventNotifyConf) (notifyOption *notify.NotifyOptions, err error) {
	conf, ok := p.extra.Events[eventName]
	if !ok {
		return
	}

	notifyUser := user
	_, ok = conf.UsersMap[user]
	if !ok {
		notifyUser = conf.DefaultUser
	}
	if notifyUser == "" {
		return
	}

	notifyOption, ok = p.extra.UserNotifyConfs[notifyUser]
	if !ok {
		err = fmt.Errorf("notify user [%s] conf not found", notifyUser)
		return
	}
	return
}
