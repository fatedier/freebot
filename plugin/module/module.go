package module

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/notify"
	"github.com/fatedier/freebot/plugin"
)

var (
	PluginName = "module"
)

func init() {
	plugin.Register(PluginName, NewModulePlugin)
}

type ModuleMap struct {
	Prefix string
	Module string
}

type Extra struct {
	LablePrefix        string            `json:"label_prefix"`
	EnableCommentRoles []string          `json:"enable_comment_roles"`
	FilePrefixMap      map[string]string `json:"file_prefix_map"`

	moduleMaps []*ModuleMap `json:"-"`
}

func (ex *Extra) Complete() {
	if ex.LablePrefix == "" {
		ex.LablePrefix = "module"
	}

	ex.moduleMaps = make([]*ModuleMap, 0)
	for prefix, moduleName := range ex.FilePrefixMap {
		ex.moduleMaps = append(ex.moduleMaps, &ModuleMap{
			Prefix: prefix,
			Module: moduleName,
		})
	}
	sort.Slice(ex.moduleMaps, func(i, j int) bool {
		return len(ex.moduleMaps[i].Prefix) > len(ex.moduleMaps[j].Prefix)
	})

	if ex.EnableCommentRoles == nil {
		ex.EnableCommentRoles = make([]string, 0)
	}
}

type ModulePlugin struct {
	*plugin.BasePlugin

	extra    Extra
	cli      client.ClientInterface
	notifier notify.NotifyInterface
}

func NewModulePlugin(cli client.ClientInterface, notifier notify.NotifyInterface, options plugin.PluginOptions) (plugin.Plugin, error) {
	p := &ModulePlugin{
		cli:      cli,
		notifier: notifier,
	}
	handlerOptions := []plugin.HandlerOptions{
		plugin.HandlerOptions{
			Events:           []string{event.EvPullRequest},
			Actions:          []string{event.ActionOpened, event.ActionReopened, event.ActionSynchronize},
			ObjectNeedParams: []int{event.ObjectNeedNumber},
			Handler:          p.handlePullRequestEvent,
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

func (p *ModulePlugin) handlePullRequestEvent(ctx *event.EventContext) (err error) {
	if len(p.extra.moduleMaps) == 0 {
		return nil
	}

	number, _ := ctx.Object.Number()

	files, err := p.cli.ListFilesByPullRequest(ctx.Ctx, ctx.Owner, ctx.Repo, number)
	if err != nil {
		log.Warn("list files by pull requests [%d] error: %v", number, err)
		return err
	}

	labelsMap := make(map[string]struct{})
	for _, file := range files {
		for _, m := range p.extra.moduleMaps {
			if strings.HasPrefix(file, m.Prefix) {
				labelsMap[p.extra.LablePrefix+"/"+m.Module] = struct{}{}
				break
			}
		}
	}

	originLabels, err := p.cli.ListLabels(ctx.Ctx, ctx.Owner, ctx.Repo, number)
	if err != nil {
		log.Warn("list labels by issue number [%d] error: %v", number, err)
		return err
	}
	originLabelsMap := make(map[string]struct{})
	for _, l := range originLabels {
		if strings.HasPrefix(l, p.extra.LablePrefix+"/") {
			originLabelsMap[l] = struct{}{}
		}
	}

	if reflect.DeepEqual(labelsMap, originLabelsMap) {
		return nil
	}

	labels := make([]string, 0, len(labelsMap))
	for name, _ := range labelsMap {
		labels = append(labels, name)
	}
	err = p.cli.DoOperation(ctx.Ctx, &client.ReplaceLabelOperation{
		Owner:              ctx.Owner,
		Repo:               ctx.Repo,
		ReplaceLabelPrefix: p.extra.LablePrefix + "/",
		Number:             number,
		Labels:             labels,
	})
	if err != nil {
		return
	}
	log.Debug("[%d] add label %v", number, labels)

	// send comment
	labelRoles := p.GetLabelRoles()
	if len(p.extra.EnableCommentRoles) > 0 && len(labelRoles) > 0 {
		content := "### Label Roles:\n"
		for _, role := range p.extra.EnableCommentRoles {
			content += fmt.Sprintf("\n#### %s", role)

			for _, label := range labels {
				content += fmt.Sprintf("\n* **%s** ", label)
				roles, ok := labelRoles[label]
				if !ok {
					continue
				}

				users, ok := roles[role]
				if !ok {
					continue
				}
				content += fmt.Sprintf("*%v*", users)
			}
		}

		err = p.cli.DoOperation(ctx.Ctx, &client.AddIssueCommentOperation{
			Owner:   ctx.Owner,
			Repo:    ctx.Repo,
			Number:  number,
			Content: content,
		})
		if err != nil {
			return
		}
	}
	return
}
