package trigger

import (
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"time"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/notify"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName = "trigger"
)

func init() {
	plugin.Register(PluginName, NewTriggerPlugin)
}

type Executor struct {
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	TimeoutS int      `json:"timeout_s"`
}

type Extra struct {
	Cmds map[string]Executor `json:"cmds"`
}

func (ex *Extra) Complete() {
	if ex.Cmds == nil {
		ex.Cmds = make(map[string]Executor)
	}
	for _, cmd := range ex.Cmds {
		if cmd.TimeoutS <= 0 {
			cmd.TimeoutS = 30
		}
	}
}

type EventInfo struct {
	EventType string   `json:"event_type"`
	Owner     string   `json:"owner"`
	Repo      string   `json:"repo"`
	Number    int      `json:"number"`
	Labels    []string `json:"labels"`
}

type TriggerPlugin struct {
	*plugin.BasePlugin

	extra    Extra
	cli      client.ClientInterface
	notifier notify.NotifyInterface
}

func NewTriggerPlugin(cli client.ClientInterface, notifier notify.NotifyInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &TriggerPlugin{
		cli:      cli,
		notifier: notifier,
	}

	handlerOptions := []plugin.HandlerOptions{
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

func (p *TriggerPlugin) handleCommentEvent(ctx *event.EventContext) (err error) {
	msg, _ := ctx.Object.Body()
	number, _ := ctx.Object.Number()
	labels, hasLabels := ctx.Object.Labels()

	cmds := p.ParseCmdsFromMsg(msg, true)
	for _, cmd := range cmds {
		cmd.Name = p.ParseCmdAlias(cmd.Name)

		executor, ok := p.extra.Cmds[cmd.Name]
		if !ok {
			continue
		}
		if executor.Command == "" {
			continue
		}

		info := &EventInfo{
			EventType: ctx.Type,
			Owner:     ctx.Owner,
			Repo:      ctx.Repo,
			Number:    number,
			Labels:    make([]string, 0),
		}
		if hasLabels {
			info.Labels = labels
		}

		buf, _ := json.Marshal(info)

		newCtx, cancel := context.WithDeadline(ctx.Ctx, time.Now().Add(time.Duration(executor.TimeoutS)*time.Second))
		defer cancel()

		args := append(executor.Args, cmd.Args...)
		process := exec.CommandContext(newCtx, executor.Command, args...)
		stdin, err := process.StdinPipe()
		if err != nil {
			log.Warn("exec [%s] error: %v", executor.Command, err)
			return err
		}

		go func() {
			defer stdin.Close()
			io.WriteString(stdin, string(buf)+"\n")
		}()

		out, err := process.CombinedOutput()
		if err != nil {
			log.Warn("exec [%s] error: %v", executor.Command, err)
			return err
		}

		if len(out) > 0 {
			err = p.cli.DoOperation(ctx.Ctx, &client.AddIssueCommentOperation{
				Owner:   ctx.Owner,
				Repo:    ctx.Repo,
				Number:  number,
				Content: string(out),
			})
			if err != nil {
				return err
			}
		}
	}
	return
}
