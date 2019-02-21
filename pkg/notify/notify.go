package notify

import (
	"context"
	"fmt"
	"reflect"

	"github.com/fatedier/freebot/pkg/notify/slack"
)

type NotifyOptions struct {
	Slack slack.SlackNotifyOptions `json:"slack"`
}

type NotifyInterface interface {
	Send(ctx context.Context, options *NotifyOptions, content string) error
}

type NotifyController struct {
}

func NewNotifyController() *NotifyController {
	return &NotifyController{}
}

func (ctl *NotifyController) Send(ctx context.Context, options *NotifyOptions, content string) (err error) {
	if options == nil {
		return
	}

	if !reflect.DeepEqual(slack.SlackNotifyOptions{}, options.Slack) {
		sender := slack.NewSlackNotify(options.Slack)
		partialErr := sender.Send(ctx, content)
		if partialErr != nil {
			err = fmt.Errorf("%v; %v", err, partialErr)
		}
	}

	return
}
