package status

import (
	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/plugin"
)

/*
	extra params:
*/

var (
	PluginName = "status"
	CmdStatus  = "status"
)

func init() {
	plugin.Register(PluginName, NewStatusPlugin)
}

/*
	example:

	{
		"init_status": "wip",
		"label_precondition": {
			"wip": [],
			"wait-review": [],
			"request-changes": [],
			"approved": [{
				"is_owner": true
			}],
			"testing": [{
				"required_labels": ["status/approved"]
			}],
			"merge-ready": [
				{
					"is_owner": true,
				},
				{
					"is_qa": true,
					"required_labels": ["status/testing"]
				}
			]
		}
	}
*/
type LabelStatus struct {
	Status        string                `json:"status"`
	Preconditions []config.Precondition `json:"preconditions"`
}

type Extra struct {
	Init               LabelStatus                      `json:"init"`
	Approved           LabelStatus                      `json:"approved"`
	LabelPreconditions map[string][]config.Precondition `json:"label_precondition"`
}

type StatusPlugin struct {
	*plugin.BasePlugin

	extra Extra
	cli   client.ClientInterface
}

func NewStatusPlugin(cli client.ClientInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &StatusPlugin{
		cli: cli,
	}

	handlerOptions := []plugin.HandlerOptions{
		plugin.HandlerOptions{
			Events:           []string{event.EvPullRequest},
			Actions:          []string{event.ActionOpened},
			ObjectNeedParams: []int{event.ObjectNeedNumber},
			Handler:          p.handlePullRequestEvent,
		},
		plugin.HandlerOptions{
			Events:           []string{event.EvPullRequestReview},
			Actions:          []string{event.ActionSubmitted},
			ObjectNeedParams: []int{event.ObjectNeedNumber, event.ObjectNeedSenderUser, event.ObjectNeedReviewState},
			Handler:          p.handlePullRequestReviewEvent,
		},
		plugin.HandlerOptions{
			Events:           []string{event.EvIssueComment, event.EvPullRequest, event.EvPullRequestReviewComment},
			Actions:          []string{event.ActionCreated},
			ObjectNeedParams: []int{event.ObjectNeedBody, event.ObjectNeedNumber},
			Handler:          p.hanldeCommentEvent,
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

func (p *StatusPlugin) hanldeCommentEvent(ctx *event.EventContext) (err error) {
	msg, _ := ctx.Object.Body()
	number, _ := ctx.Object.Number()

	cmds := p.ParseCmdsFromMsg(msg, true)
	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)

		// only attach one status label and remove old status label
		if cmd.Name == CmdStatus {
			log.Debug("cmd: %v", cmd)
			if len(cmd.Args) > 0 {
				arg := cmd.Args[0]
				arg = p.ParseLabelAlias(arg)
				preconditions, ok := p.extra.LabelPreconditions[arg]
				if !ok {
					log.Warn("status [%s] not support", arg)
					continue
				}

				// one preconditions should be satisfied
				err = p.CheckPreconditions(ctx, preconditions)
				if err != nil {
					log.Warn("all preconditions check failed: %v", err)
					return
				}

				err = p.cli.DoOperation(ctx.Ctx, &client.ReplaceLabelOperation{
					Owner:              ctx.Owner,
					Repo:               ctx.Repo,
					ReplaceLabelPrefix: CmdStatus + "/",
					Number:             number,
					Labels:             []string{CmdStatus + "/" + arg},
				})
				if err != nil {
					return
				}
				log.Debug("[%d] add label %s", number, CmdStatus+"/"+arg)
				break
			}
		}
	}
	return
}

func (p *StatusPlugin) handlePullRequestEvent(ctx *event.EventContext) (err error) {
	if p.extra.Init.Status != "" {
		number, _ := ctx.Object.Number()
		err = p.CheckPreconditions(ctx, p.extra.Init.Preconditions)
		if err != nil {
			log.Warn("init preconditions check failed: %v", err)
			return
		}

		err = p.cli.DoOperation(ctx.Ctx, &client.ReplaceLabelOperation{
			Owner:              ctx.Owner,
			Repo:               ctx.Repo,
			ReplaceLabelPrefix: CmdStatus + "/",
			Number:             number,
			Labels:             []string{CmdStatus + "/" + p.extra.Init.Status},
		})
		if err != nil {
			return
		}
		log.Debug("[%d] add label %s", number, CmdStatus+"/"+p.extra.Init.Status)
	}
	return
}

func (p *StatusPlugin) handlePullRequestReviewEvent(ctx *event.EventContext) (err error) {
	if p.extra.Approved.Status != "" {
		number, _ := ctx.Object.Number()
		reviewState, _ := ctx.Object.ReviewState()
		if reviewState != event.ReviewStateApproved {
			return
		}

		err = p.CheckPreconditions(ctx, p.extra.Approved.Preconditions)
		if err != nil {
			log.Warn("approved preconditions check failed: %v", err)
			return
		}

		err = p.cli.DoOperation(ctx.Ctx, &client.ReplaceLabelOperation{
			Owner:              ctx.Owner,
			Repo:               ctx.Repo,
			ReplaceLabelPrefix: CmdStatus + "/",
			Number:             number,
			Labels:             []string{CmdStatus + "/" + p.extra.Approved.Status},
		})
		if err != nil {
			return
		}
		log.Debug("[%d] add label %s", number, CmdStatus+"/"+p.extra.Approved.Status)
	}
	return
}
