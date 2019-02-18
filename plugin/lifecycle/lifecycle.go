package lifecycle

import (
	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName = "lifecycle"
	CmdClose   = "close"
	CmdReopen  = "reopen"
)

func init() {
	plugin.Register(PluginName, NewLifecyclePlugin)
}

type Extra struct {
}

type LifecyclePlugin struct {
	*plugin.BasePlugin

	cli client.ClientInterface
}

func NewLifecyclePlugin(cli client.ClientInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &LifecyclePlugin{
		cli: cli,
	}
	handlerOptions := []plugin.HandlerOptions{
		plugin.HandlerOptions{
			Events:           []string{event.EvIssueComment, event.EvPullRequest, event.EvPullRequestReviewComment},
			Actions:          []string{event.ActionCreated},
			ObjectNeedParams: []int{event.ObjectNeedBody, event.ObjectNeedNumber},
			Handler:          p.hanldeCommentEvent,
		},
	}
	options.Handlers = handlerOptions

	p.BasePlugin = plugin.NewBasePlugin(PluginName, options)
	return p, nil
}

func (p *LifecyclePlugin) hanldeCommentEvent(ctx *event.EventContext) (err error) {
	msg, _ := ctx.Object.Body()
	number, _ := ctx.Object.Number()

	cmds := p.ParseCmdsFromMsg(msg, false)

	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)
		switch cmd.Name {
		case CmdClose:
			err = p.cli.DoOperation(ctx.Ctx, &client.CloseOperation{
				Owner:  ctx.Owner,
				Repo:   ctx.Repo,
				Number: number,
				Object: ctx.Object,
			})
			return
		case CmdReopen:
			err = p.cli.DoOperation(ctx.Ctx, &client.ReopenOperation{
				Owner:  ctx.Owner,
				Repo:   ctx.Repo,
				Number: number,
				Object: ctx.Object,
			})
			return
		default:
		}
	}
	return
}
