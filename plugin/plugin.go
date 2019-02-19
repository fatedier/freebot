package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/notify"
)

var creators map[string]CreatorFn

func init() {
	creators = make(map[string]CreatorFn)
}

type CreatorFn func(cli client.ClientInterface, notifier notify.NotifyInterface, options PluginOptions) (Plugin, error)

type Handler func(ctx *event.EventContext) (err error)

func Register(name string, fn CreatorFn) {
	creators[name] = fn
}

func Create(cli client.ClientInterface, notifier notify.NotifyInterface, name string, options PluginOptions) (p Plugin, err error) {
	if fn, ok := creators[name]; ok {
		p, err = fn(cli, notifier, options)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", name)
	}
	return
}

type HandlerOptions struct {
	Events           []string
	Actions          []string // empty means support all
	ObjectNeedParams []int
	Handler          Handler
}

type Plugin interface {
	Name() string
	HanldeEvent(ctx *event.EventContext) (notSupport bool, err error)
}

type PluginOptions struct {
	Owner         string
	Repo          string
	Alias         config.AliasOptions
	Roles         config.RoleOptions
	Preconditions []config.Precondition
	Extra         interface{}

	// filled by plugin
	Handlers []HandlerOptions
}

func (options *PluginOptions) Complete(owner, repo string, alias config.AliasOptions,
	roles config.RoleOptions, preconditions []config.Precondition, extra interface{}) {

	options.Owner = owner
	options.Repo = repo
	options.Alias = alias
	options.Roles = roles
	options.Preconditions = preconditions
	if options.Preconditions == nil {
		options.Preconditions = make([]config.Precondition, 0)
	}
	options.Extra = extra
}

type BasePlugin struct {
	name          string
	owner         string
	repo          string
	alias         config.AliasOptions
	roles         config.RoleOptions
	preconditions []config.Precondition
	extra         interface{}

	handlers []HandlerOptions
}

func NewBasePlugin(name string, options PluginOptions) *BasePlugin {
	return &BasePlugin{
		name:          name,
		owner:         options.Owner,
		repo:          options.Repo,
		alias:         options.Alias,
		roles:         options.Roles,
		preconditions: options.Preconditions,
		extra:         options.Extra,
		handlers:      options.Handlers,
	}
}

func (p *BasePlugin) Name() string {
	return p.name
}

func (p *BasePlugin) GetOwner() string {
	return p.owner
}

func (p *BasePlugin) GetRepo() string {
	return p.repo
}

func (p *BasePlugin) GetAlias() config.AliasOptions {
	return p.alias
}

func (p *BasePlugin) GetRoles() config.RoleOptions {
	return p.roles
}

func (p *BasePlugin) GetPreconditions() []config.Precondition {
	return p.preconditions
}

func (p *BasePlugin) GetExtra() interface{} {
	return p.extra
}

func (p *BasePlugin) UnmarshalTo(v interface{}) error {
	buf, err := json.Marshal(p.extra)
	if err != nil {
		return fmt.Errorf("[%s] extra conf parse failed", p.name)
	}

	if err = json.Unmarshal(buf, &v); err != nil {
		return fmt.Errorf("[%s] extra conf parse failed", p.name)
	}
	log.Info("[%s/%s] [%s]", p.owner, p.repo, p.name)
	return nil
}

func (p *BasePlugin) IsSupported(ctx *event.EventContext, handlerOptions HandlerOptions) bool {
	if len(handlerOptions.Events) > 0 {
		if !p.IsSupportedEvent(ctx.Type, handlerOptions) {
			return false
		}
	}

	if len(handlerOptions.Actions) > 0 {
		action, ok := ctx.Object.Action()
		if !ok {
			return false
		}

		if !p.IsSupportedAction(action, handlerOptions) {
			return false
		}
	}
	return true
}

func (p *BasePlugin) IsSupportedEvent(eventType string, handlerOptions HandlerOptions) bool {
	if len(handlerOptions.Events) == 0 {
		return true
	}

	for _, e := range handlerOptions.Events {
		if e == eventType {
			return true
		}
	}
	return false
}

func (p *BasePlugin) IsSupportedAction(action string, handlerOptions HandlerOptions) bool {
	if len(handlerOptions.Actions) == 0 {
		return true
	}

	for _, v := range handlerOptions.Actions {
		if v == action {
			return true
		}
	}
	return false
}

func (p *BasePlugin) ParseCmdAlias(str string) string {
	if p.alias.Cmds != nil {
		if new, ok := p.alias.Cmds[str]; ok {
			return new
		}
	}
	return str
}

func (p *BasePlugin) ParseLabelAlias(str string) string {
	if p.alias.Labels != nil {
		if new, ok := p.alias.Labels[str]; ok {
			return new
		}
	}
	return str
}

func (p *BasePlugin) ParseUserAlias(str string) string {
	if p.alias.Users != nil {
		if new, ok := p.alias.Users[str]; ok {
			return new
		}
	}
	return str
}

func (p *BasePlugin) IsSpecifiedRoles(user string, roles []string) bool {
	for _, role := range roles {
		users, ok := p.roles[role]
		if !ok {
			return false
		}

		find := false
		for _, t := range users {
			if t == user {
				find = true
				break
			}
		}
		if !find {
			return false
		}
	}
	return true
}

func (p *BasePlugin) CheckPluginPreconditions(ctx *event.EventContext) (err error) {
	return p.CheckPreconditions(ctx, p.preconditions)
}

func (p *BasePlugin) CheckPreconditions(ctx *event.EventContext, preconditions []config.Precondition) (err error) {
	if len(preconditions) == 0 {
		return nil
	}

	var partialErr error
	for _, pre := range preconditions {
		partialErr = p.CheckPrecondition(ctx, pre)
		if partialErr == nil {
			return nil
		} else {
			err = fmt.Errorf("%v; %v", err, partialErr)
		}
	}
	return
}

func (p *BasePlugin) CheckPrecondition(ctx *event.EventContext, precondition config.Precondition) (err error) {
	if precondition.IsAuthor {
		err = p.CheckIsAuthor(ctx)
		if err != nil {
			return
		}
	}

	if len(precondition.RequiredRoles) > 0 {
		err = p.CheckRequiredRoles(ctx, precondition.RequiredRoles)
		if err != nil {
			return
		}
	}

	if len(precondition.RequiredLabels) > 0 {
		err = p.CheckRequiredLabels(ctx, precondition.RequiredLabels)
		if err != nil {
			return
		}
	}

	if len(precondition.RequiredLabelPrefix) > 0 {
		err = p.CheckRequiredLabelPrefix(ctx, precondition.RequiredLabelPrefix)
		if err != nil {
			return
		}
	}
	return nil
}

func (p *BasePlugin) CheckIsAuthor(ctx *event.EventContext) error {
	author, ok := ctx.Object.Author()
	sender, ok2 := ctx.Object.SenderUser()
	isAuthor := author == sender
	if !ok || !ok2 || !isAuthor {
		return fmt.Errorf("check is author failed, author [%s], sender [%s]", author, sender)
	}
	return nil
}

func (p *BasePlugin) CheckRequiredRoles(ctx *event.EventContext, roles []string) error {
	sender, ok := ctx.Object.SenderUser()
	if !ok {
		return fmt.Errorf("check required roles failed, get sender failed")
	}

	if !p.IsSpecifiedRoles(sender, roles) {
		return fmt.Errorf("check required roles failed: %s not in roles %v", sender, roles)
	}
	return nil
}

func (p *BasePlugin) CheckRequiredLabels(ctx *event.EventContext, labels []string) error {
	all, ok := ctx.Object.Labels()
	if !ok {
		return fmt.Errorf("check required labels failed, get labels failed")
	}

	allMap := make(map[string]struct{})
	for _, v := range all {
		allMap[v] = struct{}{}
	}

	for _, label := range labels {
		if _, ok := allMap[label]; !ok {
			return fmt.Errorf("check required labels failed: %s doesn't exist", label)
		}
	}
	return nil
}

func (p *BasePlugin) CheckRequiredLabelPrefix(ctx *event.EventContext, prefix []string) error {
	all, ok := ctx.Object.Labels()
	if !ok {
		return fmt.Errorf("check required label prefix failed: get labels failed")
	}

	for _, prefixStr := range prefix {
		hasOne := false
		for _, v := range all {
			if strings.HasPrefix(v, prefixStr) {
				hasOne = true
				break
			}
		}
		if !hasOne {
			return fmt.Errorf("check required label prefix failed: %s prefix label not found", prefixStr)
		}
	}
	return nil
}

func (p *BasePlugin) HanldeEvent(ctx *event.EventContext) (notSupport bool, err error) {
	handled := false
	meetPreconditions := false
	for _, handlerOptions := range p.handlers {
		if !p.IsSupported(ctx, handlerOptions) {
			continue
		}
		handled = true

		for _, param := range handlerOptions.ObjectNeedParams {
			var ok bool
			paramName := ""
			switch param {
			case event.ObjectNeedBody:
				_, ok = ctx.Object.Body()
				paramName = "body"
			case event.ObjectNeedNumber:
				_, ok = ctx.Object.Number()
				paramName = "number"
			case event.ObjectNeedAction:
				_, ok = ctx.Object.Action()
				paramName = "action"
			case event.ObjectNeedAuthor:
				_, ok = ctx.Object.Author()
				paramName = "author"
			case event.ObjectNeedCommentAuthor:
				_, ok = ctx.Object.CommentAuthor()
				paramName = "comment author"
			case event.ObjectNeedSenderUser:
				_, ok = ctx.Object.SenderUser()
				paramName = "sender user"
			case event.ObjectNeedLabels:
				_, ok = ctx.Object.Labels()
				paramName = "labels"
			case event.ObjectNeedReviewState:
				_, ok = ctx.Object.ReviewState()
				paramName = "review state"
			case event.ObjectNeedCheckRunStatus:
				_, ok = ctx.Object.CheckRunStatus()
				paramName = "check run status"
			case event.ObjectNeedCheckRunConclusion:
				_, ok = ctx.Object.CheckRunConclusion()
				paramName = "check run conclusion"
			case event.ObjectNeedCheckSuiteStatus:
				_, ok = ctx.Object.CheckSuiteStatus()
				paramName = "check suite status"
			case event.ObjectNeedCheckSuiteConclusion:
				_, ok = ctx.Object.CheckSuiteConclusion()
				paramName = "check suite conclusion"
			default:
				log.Error("error ObjectNeedParams setting")
				continue
			}

			if !ok {
				err = fmt.Errorf("can't get %s from payload", paramName)
				return
			}
		}

		// only check plugin preconditions once
		if !meetPreconditions {
			err = p.CheckPluginPreconditions(ctx)
			if err != nil {
				return
			}
			meetPreconditions = true
		}

		err = handlerOptions.Handler(ctx)
		if err != nil {
			return
		}
	}

	if !handled {
		log.Debug("plugin [%s] handlers not support", p.name)
		return true, nil
	}
	return false, nil
}
