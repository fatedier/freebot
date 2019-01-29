package plugin

import (
	"fmt"
	"strings"

	"github.com/fatedier/freebot/pkg/client"
	"github.com/fatedier/freebot/pkg/config"
	"github.com/fatedier/freebot/pkg/event"
)

var creators map[string]CreatorFn

func init() {
	creators = make(map[string]CreatorFn)
}

type CreatorFn func(cli client.ClientInterface, owner string, repo string, base BasePluginOptions, extra interface{}) (Plugin, error)

func Register(name string, fn CreatorFn) {
	creators[name] = fn
}

func Create(cli client.ClientInterface, name string, owner string, repo string, base BasePluginOptions, extra interface{}) (p Plugin, err error) {
	if fn, ok := creators[name]; ok {
		p, err = fn(cli, owner, repo, base, extra)
	} else {
		err = fmt.Errorf("plugin [%s] is not registered", name)
	}
	return
}

type Plugin interface {
	Name() string
	HanldeEvent(ctx *event.EventContext) error
}

type BasePluginOptions struct {
	Alias        config.AliasOptions `json:"alias"`
	Roles        config.RoleOptions  `json:"roles"`
	Precondition config.Precondition `json:"precondition"`

	Owner         string   `json:"-"`
	Repo          string   `json:"-"`
	SupportEvents []string `json:"-"`
}

type BasePlugin struct {
	alias        config.AliasOptions
	roles        config.RoleOptions
	precondition config.Precondition

	name          string
	owner         string
	repo          string
	supportEvents []string
}

func NewBasePlugin(name string, options BasePluginOptions) *BasePlugin {
	return &BasePlugin{
		alias:        options.Alias,
		roles:        options.Roles,
		precondition: options.Precondition,

		name:          name,
		owner:         options.Owner,
		repo:          options.Repo,
		supportEvents: options.SupportEvents,
	}
}

func (p *BasePlugin) Name() string {
	return p.name
}

func (p *BasePlugin) GetAlias() config.AliasOptions {
	return p.alias
}

func (p *BasePlugin) GetRoles() config.RoleOptions {
	return p.roles
}

func (p *BasePlugin) GetPrecondition() config.Precondition {
	return p.precondition
}

func (p *BasePlugin) GetOwner() string {
	return p.owner
}

func (p *BasePlugin) GetRepo() string {
	return p.repo
}

func (p *BasePlugin) IsSupportedEvent(eventType string) bool {
	for _, e := range p.supportEvents {
		if e == eventType {
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
	if p.alias.Labels != nil {
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

func (p *BasePlugin) CheckPluginPrecondition(eventCtx *event.EventContext) error {
	return p.CheckPrecondition(eventCtx, p.precondition)
}

func (p *BasePlugin) CheckPrecondition(eventCtx *event.EventContext, precondition config.Precondition) (err error) {
	if precondition.IsAuthor {
		err = p.CheckIsAuthor(eventCtx)
		if err != nil {
			return
		}
	}

	if precondition.IsOwner {
		err = p.CheckIsOwner(eventCtx)
		if err != nil {
			return
		}
	}

	if precondition.IsQA {
		err = p.CheckIsQA(eventCtx)
		if err != nil {
			return
		}
	}

	if len(precondition.RequiredLabels) > 0 {
		err = p.CheckRequiredLabels(eventCtx, precondition.RequiredLabels)
		if err != nil {
			return
		}
	}

	if len(precondition.RequiredLabelPrefix) > 0 {
		err = p.CheckRequiredLabelPrefix(eventCtx, precondition.RequiredLabelPrefix)
		if err != nil {
			return
		}
	}
	return nil
}

func (p *BasePlugin) CheckIsAuthor(eventCtx *event.EventContext) error {
	obj := client.NewObject(eventCtx.Payload)
	author, err := obj.GetAuthor()
	commentAuthor, err2 := obj.GetCommentAuthor()
	isAuthor := author == commentAuthor
	if err != nil || err2 != nil || isAuthor {
		return fmt.Errorf("check is author failed, err: %v, err2 %v, isAuthor: %v", err, err2, isAuthor)
	}
	return nil
}

func (p *BasePlugin) CheckIsOwner(eventCtx *event.EventContext) error {
	obj := client.NewObject(eventCtx.Payload)
	author, err := obj.GetCommentAuthor()
	if err != nil {
		return fmt.Errorf("check is owner failed: get comment author failed")
	}

	if !p.IsOwner(author) {
		return fmt.Errorf("check is owner failed: %s not owner", author)
	}
	return nil
}

func (p *BasePlugin) CheckIsQA(eventCtx *event.EventContext) error {
	obj := client.NewObject(eventCtx.Payload)
	author, err := obj.GetCommentAuthor()
	if err != nil {
		return fmt.Errorf("check is qa failed: get comment author failed")
	}

	if !p.IsQA(author) {
		return fmt.Errorf("check is qa failed: %s not qa", author)
	}
	return nil
}

func (p *BasePlugin) CheckRequiredLabels(eventCtx *event.EventContext, labels []string) error {
	obj := client.NewObject(eventCtx.Payload)
	all, err := obj.GetLables()
	if err != nil {
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

func (p *BasePlugin) CheckRequiredLabelPrefix(eventCtx *event.EventContext, prefix []string) error {
	obj := client.NewObject(eventCtx.Payload)
	all, err := obj.GetLables()
	if err != nil {
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
