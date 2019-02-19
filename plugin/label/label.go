package label

import (
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/notify"
	"github.com/fatedier/freebot/plugin"
)

/*
	"extra":{
		"kind":{
			"add_preconditions":[
			],
			"remove_preconditions":[
				{
					"required_roles": ["owner"]
				}
			],
			"labels": ["feature", "bug"]
		 }
	}
}
*/

const (
	PluginRemoveCmdPrefix = "remove-"
)

var (
	PluginName = "label"
)

func init() {
	plugin.Register(PluginName, NewLabelPlugin)
}

type LabelStatus struct {
	AddPreconditions    []config.Precondition `json:"add_preconditions"`
	RemovePreconditions []config.Precondition `json:"remove_preconditions"`
	Labels              []string              `json:"labels"`
}

type Extra map[string]LabelStatus

type LablePlugin struct {
	*plugin.BasePlugin
	extra Extra

	cli      client.ClientInterface
	notifier notify.NotifyInterface
}

func NewLabelPlugin(cli client.ClientInterface, notifier notify.NotifyInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &LablePlugin{
		cli:      cli,
		notifier: notifier,
	}

	handlerOptions := []plugin.HandlerOptions{
		plugin.HandlerOptions{
			Events:           []string{event.EvIssueComment, event.EvPullRequest, event.EvPullRequestReviewComment},
			Actions:          []string{event.ActionCreated},
			ObjectNeedParams: []int{event.ObjectNeedBody, event.ObjectNeedNumber, event.ObjectNeedLabels},
			Handler:          p.handleCommentEvent,
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

func (p *LablePlugin) handleCommentEvent(ctx *event.EventContext) (err error) {
	log.Debug("lable plugin extra config is: %v", p.extra)

	msg, _ := ctx.Object.Body()
	number, _ := ctx.Object.Number()

	cmds := p.ParseCmdsFromMsg(msg, false)
	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)
		var arg string
		if len(cmd.Args) > 0 {
			arg = p.ParseLabelAlias(cmd.Args[0])
		} else {
			continue
		}

		if cv, ok := p.extra[cmd.Name]; ok {
			if stringContains(cv.Labels, arg) {
				// one preconditions should be satisfied
				err = p.CheckPreconditions(ctx, p.extra[cmd.Name].AddPreconditions)
				if err != nil {
					log.Warn("all preconditions check failed: %v", err)
					return
				}

				err = p.cli.DoOperation(ctx.Ctx, &client.AddLabelOperation{
					Owner:  ctx.Owner,
					Repo:   ctx.Repo,
					Number: number,
					Labels: []string{cmd.Name + "/" + arg},
				})
				if err != nil {
					return
				}
				log.Debug("[%d] add label %s", number, cmd.Name+"/"+arg)
			}
		}

		if strings.HasPrefix(cmd.Name, PluginRemoveCmdPrefix) {
			trimName := strings.TrimPrefix(cmd.Name, PluginRemoveCmdPrefix)
			if cv, ok := p.extra[trimName]; ok {
				if stringContains(cv.Labels, arg) {
					// one preconditions should be satisfied
					err = p.CheckPreconditions(ctx, p.extra[trimName].RemovePreconditions)
					if err != nil {
						log.Warn("all preconditions check failed: %v", err)
						return
					}

					err = p.cli.DoOperation(ctx.Ctx, &client.RemoveLabelOperation{
						Owner:  ctx.Owner,
						Repo:   ctx.Repo,
						Number: number,
						Label:  trimName + "/" + arg,
					})
					if err != nil {
						return
					}
					log.Debug("[%d] remove label :%v", number, trimName+"/"+arg)
				}
			}
		}
	}
	return
}

func stringContains(strs []string, s string) bool {
	for _, v := range strs {
		if v == s {
			return true
		}
	}
	return false
}
