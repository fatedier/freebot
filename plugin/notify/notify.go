package notify

import (
	"fmt"
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
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
	CmdPing    = "ping"
)

func init() {
	plugin.Register(PluginName, NewNotifyPlugin)
}

type EventNotifyConf struct {
	DefaultUser string              `json:"default_user"`
	Users       []string            `json:"users"`
	UsersMap    map[string]struct{} `json:"-"`
}

type PingOption struct {
	Disable       bool                  `json:"disable"`
	Preconditions []config.Precondition `json:"preconditions"`
}

type Extra struct {
	UserNotifyConfs map[string]*notify.NotifyOptions `json:"user_notify_confs"`
	Ping            PingOption                       `json:"ping"`
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
		plugin.HandlerOptions{
			Events:           []string{event.EvIssueComment, event.EvPullRequest, event.EvPullRequestReviewComment},
			Actions:          []string{event.ActionCreated},
			ObjectNeedParams: []int{event.ObjectNeedBody, event.ObjectNeedCommentAuthor, event.ObjectNeedIssueHTMLURL},
			Handler:          p.handleCommentEvent,
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

func (p *NotifyPlugin) handleCommentEvent(ctx *event.EventContext) (err error) {
	if p.extra.Ping.Disable {
		return nil
	}

	msg, _ := ctx.Object.Body()
	author, _ := ctx.Object.CommentAuthor()
	issueHTMLURL, _ := ctx.Object.IssueHTMLURL()

	err = p.CheckPreconditions(ctx, p.extra.Ping.Preconditions)
	if err != nil {
		log.Warn("preconditions check failed: %v", err)
		return
	}

	cmds := p.ParseCmdsFromMsg(msg, false)
	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)

		additionalMsg := ""
		switch cmd.Name {
		case CmdPing:
			log.Debug("ping event")
			if len(cmd.Args) == 0 {
				continue
			}

			if len(cmd.Args) > 1 {
				additionalMsg = strings.Join(cmd.Args[1:], " ")
			}
			user := p.ParseUserAlias(strings.TrimLeft(cmd.Args[0], "@"))

			notifyOption, ok := p.extra.UserNotifyConfs[user]
			if !ok {
				err = fmt.Errorf("notify user [%s] conf not found", user)
				return
			}

			content := fmt.Sprintf("You are pinged by [%s]", author)
			content += fmt.Sprintf("\n%s", issueHTMLURL)
			if additionalMsg != "" {
				content += fmt.Sprintf("\n%s", additionalMsg)
			}
			//log.Debug("check run [%s] [%s] [%s], send notify", pr.Title, run.Status, run.Conclusion)
			err = p.notifier.Send(ctx.Ctx, notifyOption, content)
		}
	}
	return
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
