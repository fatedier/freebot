package event

import (
	"context"

	"github.com/fatedier/freebot/pkg/client"
)

const (
	EvIssueComment             = "issue_comment"
	EvPullRequest              = "pull_request"
	EvPullRequestReviewComment = "pull_request_review_comment"
	EvPing                     = "ping"
)

const (
	ActionCreated = "created"
	ActionDeleted = "deleted"
)

const (
	ObjectNeedBody = iota
	ObjectNeedNumber
	ObjectNeedAction
	ObjectNeedAuthor
	ObjectNeedCommentAuthor
	ObjectNeedLabels
)

type EventContext struct {
	Ctx    context.Context
	Type   string
	Owner  string
	Repo   string
	Object *client.Object
}
