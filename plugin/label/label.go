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
	extra params:
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
	Preconditions []config.Precondition
	Labels        []string
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
				log.Debug("[%d] add label %s", number, cmd, "kind/feature")
			}
		}

		if strings.HasPrefix(cmd.Name, PluginRemoveCmdPrefix) {
			trimName := strings.TrimPrefix(cmd.Name, PluginRemoveCmdPrefix)
			if cv, ok := p.extra[trimName]; ok {
				if len(cmd.Args) > 0 && stringContains(cv.Labels, cmd.Args[0]) {
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

func stringContains(strs []string, s string) bool {
	for _, v := range strs {
		if v == s {
			return true
		}
	}
	return false
}
