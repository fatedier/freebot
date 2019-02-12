package event

import (
	"context"

	"github.com/fatedier/freebot/pkg/client"
)

const (
	EvIssueComment             = "issue_comment"
	EvPullRequest              = "pull_request"
	EvPullRequestReview        = "pull_request_review"
	EvPullRequestReviewComment = "pull_request_review_comment"
	EvPing                     = "ping"
)

const (
	ActionCreated              = "created"
	ActionOpened               = "opened"
	ActionSubmitted            = "submitted"
	ActionDeleted              = "deleted"
	ActionClosed               = "closed"
	ActionLabeled              = "labeled"
	ActionUnlabeled            = "unlabeled"
	ActionReviewRequested      = "review_requested"
	ActionReviewRequestRemoved = "review_request_removed"
)

const (
	ReviewStateCommented        = "commented"
	ReviewStateApproved         = "approved"
	ReviewStateChangesRequested = "changes_requested"
)

const (
	ObjectNeedBody = iota
	ObjectNeedNumber
	ObjectNeedAction
	ObjectNeedAuthor
	ObjectNeedCommentAuthor
	ObjectNeedSenderUser
	ObjectNeedLabels
	ObjectNeedReviewState
)

type EventContext struct {
	Ctx    context.Context
	Type   string
	Owner  string
	Repo   string
	Object *client.Object
}
