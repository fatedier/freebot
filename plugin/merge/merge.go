package merge

import (
	"fmt"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/notify"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName = "merge"
	CmdMerge   = "merge"
)

func init() {
	plugin.Register(PluginName, NewMergePlugin)
}

type Extra struct {
}

type MergePlugin struct {
	*plugin.BasePlugin

	cli      client.ClientInterface
	notifier notify.NotifyInterface
}

func NewMergePlugin(cli client.ClientInterface, notifier notify.NotifyInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &MergePlugin{
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
	return p, nil
}

func (p *MergePlugin) handleCommentEvent(ctx *event.EventContext) (err error) {
	msg, _ := ctx.Object.Body()
	number, _ := ctx.Object.Number()

	cmds := p.ParseCmdsFromMsg(msg, false)
	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)

		if cmd.Name == CmdMerge {
			var mergeable bool
			mergeable, err = p.cli.CheckMergeable(ctx.Ctx, ctx.Owner, ctx.Repo, number)
			if err != nil {
				return
			}

			if !mergeable {
				err = fmt.Errorf("[%s] pull request not mergeable", PluginName)
				return
			}

			err = p.cli.DoOperation(ctx.Ctx, &client.MergeOperation{
				Owner:  ctx.Owner,
				Repo:   ctx.Repo,
				Number: number,
			})
			return
		}
	}
	return
}
