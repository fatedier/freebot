package event

import (
	"context"
)

const (
	EvIssueComment             = "issue_comment"
	EvPullRequest              = "pull_request"
	EvPullRequestReviewComment = "pull_request_review_comment"
)

type EventContext struct {
	Ctx     context.Context
	Type    string
	Owner   string
	Repo    string
	Payload interface{}
}
