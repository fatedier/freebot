package status

import (
	"encoding/json"
	"fmt"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/plugin"
)

/*
	extra params:
*/

var (
	PluginName    = "status"
	SupportEvents = []string{event.EvIssueComment, event.EvPullRequest}
	CmdStatus     = "status"
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
	LabelPreconditions map[string][]config.Precondition
}

type StatusPlugin struct {
	*plugin.BasePlugin

	extra Extra
	cli   client.ClientInterface
}

func NewStatusPlugin(cli client.ClientInterface, owner string, repo string, base plugin.BasePluginOptions, extra interface{}) (plugin.Plugin, error) {
	base.Owner = owner
	base.Repo = repo
	base.SupportEvents = SupportEvents
	p := &StatusPlugin{
		cli: cli,
	}
	p.BasePlugin = plugin.NewBasePlugin(PluginName, base)

	buf, err := json.Marshal(extra)
	if err != nil {
		return nil, fmt.Errorf("[%s] extra conf parse failed", PluginName)
	}

	if err = json.Unmarshal(buf, &p.extra); err != nil {
		return nil, fmt.Errorf("[%s] extra conf parse failed", PluginName)
	}
	return p, nil
}

func (p *StatusPlugin) HanldeEvent(ctx *event.EventContext) (err error) {
	var (
		msg    string
		number int
	)
	obj := client.NewObject(ctx.Payload)
	msg, err = obj.GetBody()
	if err != nil {
		return
	}

	number, err = obj.GetNumber()
	if err != nil {
		return
	}

	cmds := p.ParseCmdsFromMsg(msg, true)
	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)

		// only attach one status label and remove old status label
		if cmd.Name == CmdStatus {
			if len(cmd.Args) > 0 {
				arg := cmd.Args[0]
				arg = p.ParseLabelAlias(arg)
				preconditions, ok := p.extra.LabelPreconditions[arg]
				if !ok {
					continue
				}

				for _, precondition := range preconditions {
					err = p.CheckPrecondition(ctx, precondition)
					if err != nil {
						return
					}
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
				break
			}
		}
	}
	return
}
