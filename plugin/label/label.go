package label

import (
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
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

	cli client.ClientInterface
}

func NewLabelPlugin(cli client.ClientInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &LablePlugin{
		cli: cli,
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
		if cv, ok := p.extra[cmd.Name]; ok {
			if len(cmd.Args) > 0 && stringContains(cv.Labels, cmd.Args[0]) {
				err = p.cli.DoOperation(ctx.Ctx, &client.AddLabelOperation{
					Owner:  ctx.Owner,
					Repo:   ctx.Repo,
					Number: number,
					Labels: []string{cmd.Name + "/" + cmd.Args[0]},
				})
				if err != nil {
					return
				}
				log.Debug("[%d] add label %s", number, cmd.Name+"/"+cmd.Args[0])
			}
		}

		if strings.HasPrefix(cmd.Name, PluginRemoveCmdPrefix) {
			trimName := strings.TrimPrefix(cmd.Name, PluginRemoveCmdPrefix)
			if cv, ok := p.extra[trimName]; ok {
				if len(cmd.Args) > 0 && stringContains(cv.Labels, cmd.Args[0]) {
					// one preconditions should be satisfied
					err = p.checkRemoveConditions(ctx, p.extra[trimName].RemovePreconditions)
					if err != nil {
						log.Warn("all preconditions check failed: %v", err)
						return
					}

					err = p.cli.DoOperation(ctx.Ctx, &client.RemoveLabelOperation{
						Owner:  ctx.Owner,
						Repo:   ctx.Repo,
						Number: number,
						Label:  trimName + "/" + cmd.Args[0],
					})
					if err != nil {
						return
					}
					log.Debug("[%d] remove label :%v", number, trimName+"/"+cmd.Args[0])
				}
			}
		}
	}
	return
}

func (p *LablePlugin) checkAddCondition(ctx *event.EventContext, preconditions []config.Precondition) error {
	return nil
}

func (p *LablePlugin) checkRemoveConditions(ctx *event.EventContext, preconditions []config.Precondition) error {
	// one preconditions should be satisfied
	err := p.CheckPreconditions(ctx, preconditions)
	if err != nil {
		return err
	}
	return nil
}

func stringContains(strs []string, s string) bool {
	for _, v := range strs {
		if v == s {
			return true
		}
	}
	return false
}
