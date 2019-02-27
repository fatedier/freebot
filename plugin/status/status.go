package status

import (
	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/notify"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName = "status"
	CmdStatus  = "status"

	SupportTriggers = map[string]struct{}{
		"pull_request/opened":      struct{}{},
		"pull_request/synchronize": struct{}{},
		"pull_request/labeled":     struct{}{},
		"pull_request/unlabeled":   struct{}{},

		"pull_request_review/submitted/approved":          struct{}{},
		"pull_request_review/submitted/commented":         struct{}{},
		"pull_request_review/submitted/changes_requested": struct{}{},
	}
)

func init() {
	plugin.Register(PluginName, NewStatusPlugin)
}

type LabelStatus struct {
	Status        string                `json:"status"`
	Preconditions []config.Precondition `json:"preconditions"`
}

type Extra struct {
	// key should be in SupportTriggers
	EventsTrigger map[string][]LabelStatus `json:"events_trigger"`

	LabelPreconditions map[string][]config.Precondition `json:"label_precondition"`
}

func (ex *Extra) Complete() {
	if ex.EventsTrigger == nil {
		ex.EventsTrigger = make(map[string][]LabelStatus)
	}

	if ex.LabelPreconditions == nil {
		ex.LabelPreconditions = make(map[string][]config.Precondition)
	}
}

type StatusPlugin struct {
	*plugin.BasePlugin

	extra    Extra
	cli      client.ClientInterface
	notifier notify.NotifyInterface
}

func NewStatusPlugin(cli client.ClientInterface, notifier notify.NotifyInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &StatusPlugin{
		cli:      cli,
		notifier: notifier,
	}

	handlerOptions := []plugin.HandlerOptions{
		plugin.HandlerOptions{
			Events:  []string{event.EvPullRequest},
			Actions: []string{event.ActionOpened, event.ActionSynchronize, event.ActionLabeled, event.ActionUnlabeled},
			ObjectNeedParams: []int{event.ObjectNeedNumber, event.ObjectNeedAction, event.ObjectNeedSenderUser,
				event.ObjectNeedLabels},
			Handler: p.handlePullRequestEvent,
		},
		plugin.HandlerOptions{
			Events:  []string{event.EvPullRequestReview},
			Actions: []string{event.ActionSubmitted},
			ObjectNeedParams: []int{event.ObjectNeedNumber, event.ObjectNeedAction, event.ObjectNeedSenderUser,
				event.ObjectNeedLabels, event.ObjectNeedReviewState},
			Handler: p.handlePullRequestReviewEvent,
		},
		plugin.HandlerOptions{
			Events:           []string{event.EvIssueComment, event.EvPullRequest, event.EvPullRequestReviewComment},
			Actions:          []string{event.ActionCreated},
			ObjectNeedParams: []int{event.ObjectNeedBody, event.ObjectNeedNumber},
			Handler:          p.handleCommentEvent,
		},
	}
	options.Handlers = handlerOptions

	p.BasePlugin = plugin.NewBasePlugin(PluginName, options)

	err := p.UnmarshalTo(&p.extra)
	if err != nil {
		return nil, err
	}
	p.extra.Complete()
	return p, nil
}

func (p *StatusPlugin) handleCommentEvent(ctx *event.EventContext) (err error) {
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
	action, _ := ctx.Object.Action()
	triggerName := ctx.Type + "/" + action
	return p.handleTrigger(ctx, triggerName)
}

func (p *StatusPlugin) handlePullRequestReviewEvent(ctx *event.EventContext) (err error) {
	action, _ := ctx.Object.Action()
	state, _ := ctx.Object.ReviewState()
	triggerName := ctx.Type + "/" + action + "/" + state
	return p.handleTrigger(ctx, triggerName)
}

func (p *StatusPlugin) handleTrigger(ctx *event.EventContext, triggerName string) (err error) {
	if _, ok := SupportTriggers[triggerName]; !ok {
		return
	}

	statusPreconditions, ok := p.extra.EventsTrigger[triggerName]
	if !ok {
		return
	}

	number, _ := ctx.Object.Number()
	for _, labelStatus := range statusPreconditions {
		if labelStatus.Status == "" {
			continue
		}

		err = p.CheckPreconditions(ctx, labelStatus.Preconditions)
		if err != nil {
			log.Debug("[%d] [%s] [%s] preconditions not satisfy: %v", number, triggerName, labelStatus.Status, err)
			continue
		}

		err = p.cli.DoOperation(ctx.Ctx, &client.ReplaceLabelOperation{
			Owner:              ctx.Owner,
			Repo:               ctx.Repo,
			ReplaceLabelPrefix: CmdStatus + "/",
			Number:             number,
			Labels:             []string{CmdStatus + "/" + labelStatus.Status},
		})
		if err != nil {
			return
		}
		log.Debug("[%d] [%s] [%s] add label %s", number, triggerName, labelStatus.Status, CmdStatus+"/"+labelStatus.Status)
		break
	}
	return nil
}
