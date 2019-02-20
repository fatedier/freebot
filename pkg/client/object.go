package client

import (
	"fmt"

	"github.com/google/go-github/github"
)

type Object struct {
	payload interface{}

	hasAuthor bool
	author    string

	hasBody bool
	body    string

	hasCommentAuthor bool
	commentAuthor    string

	hasSenderUser bool
	senderUser    string

	hasNumber bool
	number    int

	hasAction bool
	action    string

	hasLabels bool
	labels    []string

	hasReviewState bool
	reviewState    string

	hasCheckEvent bool
	checkEvent    *CheckEvent
}

func NewObject(payload interface{}) *Object {
	obj := &Object{
		payload: payload,
	}

	var err error
	if obj.author, err = obj.GetAuthor(); err == nil {
		obj.hasAuthor = true
	}

	if obj.body, err = obj.GetBody(); err == nil {
		obj.hasBody = true
	}

	if obj.commentAuthor, err = obj.GetCommentAuthor(); err == nil {
		obj.hasCommentAuthor = true
	}

	if obj.senderUser, err = obj.GetSenderUser(); err == nil {
		obj.hasSenderUser = true
	}

	if obj.number, err = obj.GetNumber(); err == nil {
		obj.hasNumber = true
	}

	if obj.action, err = obj.GetAction(); err == nil {
		obj.hasAction = true
	}

	if obj.labels, err = obj.GetLables(); err == nil {
		obj.hasLabels = true
	}

	if obj.reviewState, err = obj.GetReviewState(); err == nil {
		obj.hasReviewState = true
	}

	if obj.checkEvent, err = obj.GetCheckEvent(); err == nil {
		obj.hasCheckEvent = true
	}
	return obj
}

func (obj *Object) Payload() interface{} {
	return obj.payload
}

func (obj *Object) Author() (author string, ok bool) {
	return obj.author, obj.hasAuthor
}

func (obj *Object) CommentAuthor() (author string, ok bool) {
	return obj.commentAuthor, obj.hasCommentAuthor
}

func (obj *Object) SenderUser() (user string, ok bool) {
	return obj.senderUser, obj.hasSenderUser
}

func (obj *Object) Body() (body string, ok bool) {
	return obj.body, obj.hasBody
}

func (obj *Object) Number() (number int, ok bool) {
	return obj.number, obj.hasNumber
}

func (obj *Object) Action() (action string, ok bool) {
	return obj.action, obj.hasAction
}

func (obj *Object) Labels() (labels []string, ok bool) {
	return obj.labels, obj.hasLabels
}

func (obj *Object) ReviewState() (state string, ok bool) {
	return obj.reviewState, obj.hasReviewState
}

func (obj *Object) CheckEvent() (event *CheckEvent, ok bool) {
	return obj.checkEvent, obj.hasCheckEvent
}

func (obj *Object) GetAuthor() (author string, err error) {
	switch v := obj.payload.(type) {
	case GetIssueInterface:
		author = v.GetIssue().GetUser().GetLogin()
	case GetPullRequestInterface:
		v.GetPullRequest().GetUser().GetLogin()
	default:
		err = fmt.Errorf("can't get author from payload")
		return
	}
	return
}

func (obj *Object) GetCommentAuthor() (author string, err error) {
	switch v := obj.payload.(type) {
	case *github.PullRequestEvent:
		author = v.GetPullRequest().GetUser().GetLogin()
	case GetCommentInterface:
		author = v.GetComment().GetUser().GetLogin()
	default:
		err = fmt.Errorf("can't get comment author from payload")
		return
	}
	return
}

func (obj *Object) GetSenderUser() (user string, err error) {
	switch v := obj.payload.(type) {
	case GetSenderInterface:
		user = v.GetSender().GetLogin()
	default:
		err = fmt.Errorf("can't get sender from payload")
		return
	}
	return
}

func (obj *Object) GetBody() (body string, err error) {
	switch v := obj.payload.(type) {
	case *github.IssueCommentEvent:
		body = v.GetComment().GetBody()
	case *github.PullRequestEvent:
		body = v.GetPullRequest().GetBody()
	case *github.PullRequestReviewCommentEvent:
		body = v.GetComment().GetBody()
	default:
		err = fmt.Errorf("can't get msg from payload")
		return
	}
	return
}

func (obj *Object) GetNumber() (number int, err error) {
	switch v := obj.payload.(type) {
	case GetIssueInterface:
		number = v.GetIssue().GetNumber()
	case GetPullRequestInterface:
		number = v.GetPullRequest().GetNumber()
	default:
		err = fmt.Errorf("can't get number from payload")
	}
	return
}

func (obj *Object) GetAction() (action string, err error) {
	switch v := obj.payload.(type) {
	case GetActionInterface:
		action = v.GetAction()
	default:
		err = fmt.Errorf("can't get action from payload")
	}
	return
}

func (obj *Object) GetLables() (labels []string, err error) {
	var (
		out      []github.Label
		outPoint []*github.Label
	)
	switch v := obj.payload.(type) {
	case GetIssueInterface:
		out = v.GetIssue().Labels
	case GetPullRequestInterface:
		outPoint = v.GetPullRequest().Labels
	default:
		err = fmt.Errorf("can't get labels from payload")
		return
	}

	labels = make([]string, 0, len(out))
	for _, v := range out {
		labels = append(labels, v.GetName())
	}
	for _, v := range outPoint {
		labels = append(labels, v.GetName())
	}
	return
}

func (obj *Object) GetReviewState() (state string, err error) {
	switch v := obj.payload.(type) {
	case GetReviewInterface:
		state = v.GetReview().GetState()
	default:
		err = fmt.Errorf("can't get review from payload")
		return
	}
	return
}

func (obj *Object) GetCheckEvent() (event *CheckEvent, err error) {
	event = &CheckEvent{}

	switch v := obj.payload.(type) {
	case *github.CheckRunEvent:
		event.Action = v.GetAction()
		event.IsCheckRun = true
		event.Run = &CheckRun{
			ID:         v.GetCheckRun().GetID(),
			HeadSHA:    v.GetCheckRun().GetHeadSHA(),
			Status:     v.GetCheckRun().GetStatus(),
			Conclusion: v.GetCheckRun().GetConclusion(),

			Suite: &CheckSuite{
				ID:         v.GetCheckRun().GetCheckSuite().GetID(),
				HeadSHA:    v.GetCheckRun().GetCheckSuite().GetHeadSHA(),
				Status:     v.GetCheckRun().GetCheckSuite().GetStatus(),
				Conclusion: v.GetCheckRun().GetCheckSuite().GetConclusion(),
			},
		}
	case *github.CheckSuiteEvent:
		event.Action = v.GetAction()
		event.IsCheckSuite = true
		event.Suite = &CheckSuite{
			ID:         v.GetCheckSuite().GetID(),
			HeadSHA:    v.GetCheckSuite().GetHeadSHA(),
			Status:     v.GetCheckSuite().GetStatus(),
			Conclusion: v.GetCheckSuite().GetConclusion(),
		}
	default:
		err = fmt.Errorf("can't get check event from payload")
		return
	}
	return
}

type GetActionInterface interface {
	GetAction() string
}

type GetIssueInterface interface {
	GetIssue() *github.Issue
}

type GetPullRequestInterface interface {
	GetPullRequest() *github.PullRequest
}

type GetCommentInterface interface {
	GetComment() *github.IssueComment
}

type GetNumberInterface interface {
	GetNumber() int
}

type GetRepoInterface interface {
	GetRepo() *github.Repository
}

type GetSenderInterface interface {
	GetSender() *github.User
}

type GetReviewInterface interface {
	GetReview() *github.PullRequestReview
}
