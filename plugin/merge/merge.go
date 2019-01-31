package merge

import (
	"fmt"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName       = "merge"
	SupportEvents    = []string{event.EvIssueComment, event.EvPullRequest, event.EvPullRequestReviewComment}
	SupportActions   = []string{event.ActionCreated}
	ObjectNeedParams = []int{event.ObjectNeedBody, event.ObjectNeedNumber, event.ObjectNeedLabels}
	CmdMerge         = "merge"
)

func init() {
	plugin.Register(PluginName, NewMergePlugin)
}

type Extra struct {
}

type MergePlugin struct {
	*plugin.BasePlugin

	cli client.ClientInterface
}

func NewMergePlugin(cli client.ClientInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &MergePlugin{
		cli: cli,
	}
	options.SupportEvents = SupportEvents
	options.SupportActions = SupportActions
	options.Handler = p.hanldeEvent

	p.BasePlugin = plugin.NewBasePlugin(PluginName, options)
	return p, nil
}

func (p *MergePlugin) hanldeEvent(ctx *event.EventContext) (notSupport bool, err error) {
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
