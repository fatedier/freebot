package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackNotifyOptions struct {
	Url            string `json:"url"`
	SenderUsername string `json:"sender_username"`
	Channel        string `json:"channel"`
}

type RequestPayload struct {
	Text     string `json:"text"`
	Username string `json:"username,omitempty"`
	Channel  string `json:"channel,omitempty"`
}

type SlackNotify struct {
	url      string
	username string
	channel  string
}

func NewSlackNotify(options SlackNotifyOptions) *SlackNotify {
	if options.SenderUsername == "" {
		options.SenderUsername = "freebot"
	}
	return &SlackNotify{
		url:      options.Url,
		username: options.SenderUsername,
		channel:  options.Channel,
	}
}

func (notify *SlackNotify) Send(ctx context.Context, content string) error {
	payload := &RequestPayload{
		Text:     content,
		Username: notify.username,
		Channel:  notify.channel,
	}
	buf, _ := json.Marshal(payload)
	resp, err := http.Post(notify.url, "application/json", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("slack response error code: %d", resp.StatusCode)
	}
	return nil
}
