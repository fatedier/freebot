package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
	"github.com/fatedier/freebot/pkg/log"
)

var creators map[string]CreatorFn

func init() {
	creators = make(map[string]CreatorFn)
}

type CreatorFn func(cli client.ClientInterface, options PluginOptions) (Plugin, error)

type Handler func(ctx *event.EventContext) (notSupport bool, err error)

func Register(name string, fn CreatorFn) {
	creators[name] = fn
}

func Create(cli client.ClientInterface, name string, options PluginOptions) (p Plugin, err error) {
	if fn, ok := creators[name]; ok {
		p, err = fn(cli, options)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", name)
	}
	return
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
	SupportEvents    []string
	SupportActions   []string
	ObjectNeedParams []int
	Handler          Handler
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

	supportEvents    []string
	supportActions   []string // empty means support all
	objectNeedParams []int
	handler          Handler
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

		supportEvents:    options.SupportEvents,
		supportActions:   options.SupportActions,
		objectNeedParams: options.ObjectNeedParams,
		handler:          options.Handler,
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
	log.Info("[%s/%s] [%s] extra conf: %v", p.owner, p.repo, p.name, v)
	return nil
}

func (p *BasePlugin) IsSupported(ctx *event.EventContext) bool {
	if len(p.supportEvents) > 0 {
		if !p.IsSupportedEvent(ctx.Type) {
			return false
		}
	}

	if len(p.supportActions) > 0 {
		action, ok := ctx.Object.Action()
		if !ok {
			return false
		}

		if !p.IsSupportedAction(action) {
			return false
		}
	}
	return true
}

func (p *BasePlugin) IsSupportedEvent(eventType string) bool {
	if len(p.supportEvents) == 0 {
		return true
	}

	for _, e := range p.supportEvents {
		if e == eventType {
			return true
		}
	}
	return false
}

func (p *BasePlugin) IsSupportedAction(action string) bool {
	if len(p.supportActions) == 0 {
		return true
	}

	for _, v := range p.supportActions {
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

func (p *BasePlugin) IsOwner(user string) bool {
	for _, v := range p.roles.Owner {
		if v == user {
			return true
		}
	}
	return false
}

func (p *BasePlugin) IsQA(user string) bool {
	for _, v := range p.roles.QA {
		if v == user {
			return true
		}
	}
	return false
}

func (p *BasePlugin) CheckPluginPreconditions(ctx *event.EventContext) (err error) {
	if len(p.preconditions) == 0 {
		return nil
	}

	var partialErr error
	for _, pre := range p.preconditions {
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

	if precondition.IsOwner {
		err = p.CheckIsOwner(ctx)
		if err != nil {
			return
		}
	}

	if precondition.IsQA {
		err = p.CheckIsQA(ctx)
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
	commentAuthor, ok2 := ctx.Object.CommentAuthor()
	isAuthor := author == commentAuthor
	if !ok || !ok2 || !isAuthor {
		return fmt.Errorf("check is author failed, author [%s], commentAuthor [%s]", author, commentAuthor)
	}
	return nil
}

func (p *BasePlugin) CheckIsOwner(ctx *event.EventContext) error {
	commentAuthor, ok := ctx.Object.CommentAuthor()
	if !ok {
		return fmt.Errorf("check is owner failed: get comment author failed")
	}

	if !p.IsOwner(commentAuthor) {
		return fmt.Errorf("check is owner failed: %s not owner", commentAuthor)
	}
	return nil
}

func (p *BasePlugin) CheckIsQA(ctx *event.EventContext) error {
	commentAuthor, ok := ctx.Object.CommentAuthor()
	if !ok {
		return fmt.Errorf("check is qa failed: get comment author failed")
	}

	if !p.IsQA(commentAuthor) {
		return fmt.Errorf("check is qa failed: %s not qa", commentAuthor)
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
	if !p.IsSupported(ctx) {
		log.Debug("plugin [%s] type [%s] or action not support", p.name, ctx.Type)
		return true, nil
	}

	for _, param := range p.objectNeedParams {
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
		case event.ObjectNeedLabels:
			_, ok = ctx.Object.Labels()
			paramName = "labels"
		default:
			log.Error("error ObjectNeedParams setting")
			continue
		}

		if !ok {
			err = fmt.Errorf("can't get %s from payload", paramName)
			return
		}
	}

	err = p.CheckPluginPreconditions(ctx)
	if err != nil {
		return
	}
	return p.handler(ctx)
}
