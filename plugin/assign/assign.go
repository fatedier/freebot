package assign

import (
	"fmt"
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName  = "assign"
	CmdCC       = "cc"
	CmdUnCC     = "uncc"
	CmdAssign   = "assign"
	CmdUnAssign = "unassign"
)

func init() {
	plugin.Register(PluginName, NewAssignPlugin)
}

type Extra struct {
}

type AssignPlugin struct {
	*plugin.BasePlugin

	cli client.ClientInterface
}

func NewAssignPlugin(cli client.ClientInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &AssignPlugin{
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

func (p *AssignPlugin) hanldeCommentEvent(ctx *event.EventContext) (err error) {
	msg, _ := ctx.Object.Body()
	number, _ := ctx.Object.Number()

	cmds := p.ParseCmdsFromMsg(msg, false)
	ccUsers := make([]string, 0)
	unccUsers := make([]string, 0)
	assignUsers := make([]string, 0)
	unassignUsers := make([]string, 0)

	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)
		if len(cmd.Args) > 0 {
			for _, arg := range cmd.Args {
				arg = p.ParseUserAlias(strings.TrimLeft(arg, "@"))
				log.Debug("cmd [%s], user [%s]", cmd.Name, arg)

				switch cmd.Name {
				case CmdCC:
					ccUsers = append(ccUsers, arg)
				case CmdUnCC:
					unccUsers = append(unccUsers, arg)
				case CmdAssign:
					assignUsers = append(assignUsers, arg)
				case CmdUnAssign:
					unassignUsers = append(unassignUsers, arg)
				}
			}
		}
	}

	if len(ccUsers) > 0 {
		partialErr := p.cli.DoOperation(ctx.Ctx, &client.RequestReviewsOperation{
			Owner:     ctx.Owner,
			Repo:      ctx.Repo,
			Number:    number,
			Reviewers: ccUsers,
		})
		if partialErr != nil {
			log.Warn("plugin [%s] do cc operation error: %v", PluginName, partialErr)
			err = fmt.Errorf("%v;%v", err, partialErr)
		}
	}

	if len(unccUsers) > 0 {
		partialErr := p.cli.DoOperation(ctx.Ctx, &client.RequestReviewsCancelOperation{
			Owner:           ctx.Owner,
			Repo:            ctx.Repo,
			Number:          number,
			CancelReviewers: unccUsers,
		})
		if partialErr != nil {
			log.Warn("plugin [%s] do uncc operation error: %v", PluginName, partialErr)
			err = fmt.Errorf("%v;%v", err, partialErr)
		}
	}

	if len(assignUsers) > 0 {
		partialErr := p.cli.DoOperation(ctx.Ctx, &client.AddAssignOperation{
			Owner:     ctx.Owner,
			Repo:      ctx.Repo,
			Number:    number,
			Assignees: assignUsers,
		})
		if partialErr != nil {
			log.Warn("plugin [%s] do assign operation error: %v", PluginName, partialErr)
			err = fmt.Errorf("%v;%v", err, partialErr)
		}
	}

	if len(unassignUsers) > 0 {
		partialErr := p.cli.DoOperation(ctx.Ctx, &client.RemoveAssignOperation{
			Owner:     ctx.Owner,
			Repo:      ctx.Repo,
			Number:    number,
			Assignees: unassignUsers,
		})
		if partialErr != nil {
			log.Warn("plugin [%s] do unassign operation error: %v", PluginName, partialErr)
			err = fmt.Errorf("%v;%v", err, partialErr)
		}
	}
	return
}
