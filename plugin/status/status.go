package status

import (
	"fmt"

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
	PluginName       = "status"
	SupportEvents    = []string{event.EvIssueComment, event.EvPullRequest}
	SupportActions   = []string{event.ActionCreated}
	ObjectNeedParams = []int{event.ObjectNeedBody, event.ObjectNeedNumber}
	CmdStatus        = "status"
)

func init() {
	plugin.Register(PluginName, NewStatusPlugin)
}

/*
	example:

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
*/
type Extra struct {
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
	options.SupportEvents = SupportEvents
	options.SupportActions = SupportActions
	options.ObjectNeedParams = ObjectNeedParams
	options.Handler = p.hanldeEvent

	p.BasePlugin = plugin.NewBasePlugin(PluginName, options)

	err := p.UnmarshalTo(&p.extra)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *StatusPlugin) hanldeEvent(ctx *event.EventContext) (notSupport bool, err error) {
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

				allCheckFailed := true
				if len(preconditions) == 0 {
					allCheckFailed = false
				}
				// one preconditions should be satisfied
				for _, precondition := range preconditions {
					err = p.CheckPrecondition(ctx, precondition)
					if err != nil {
						log.Debug("precondition check failed: %v", err)
					} else {
						allCheckFailed = false
					}
				}
				if allCheckFailed {
					err = fmt.Errorf("all preconditions check failed")
					log.Warn("%v", err)
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
