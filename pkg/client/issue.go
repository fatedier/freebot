package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/github"
)

type ReplaceLabelOperation struct {
	Owner              string
	Repo               string
	Number             int
	ReplaceLabelPrefix string
	Labels             []string
}

func (cli *githubClient) doReplaceLabelOperation(ctx context.Context, op *ReplaceLabelOperation) error {
	oldLabels, _, err := cli.client.Issues.ListLabelsByIssue(ctx, op.Owner, op.Repo, op.Number, &github.ListOptions{})
	if err != nil {
		return err
	}

	newLabels := make([]string, 0, len(op.Labels))
	// add new labels
	for _, name := range op.Labels {
		newLabels = append(newLabels, name)
	}

	// remove old labels with specified label prefix
	for _, l := range oldLabels {
		if !strings.HasPrefix(l.GetName(), op.ReplaceLabelPrefix) {
			newLabels = append(newLabels, l.GetName())
		}
	}

	_, _, err = cli.client.Issues.ReplaceLabelsForIssue(ctx, op.Owner, op.Repo, op.Number, newLabels)
	if err != nil {
		return err
	}
	return nil
}

type AddLabelOperation struct {
	Owner  string
	Repo   string
	Number int
	Labels []string
}

func (cli *githubClient) doAddLabelOperation(ctx context.Context, op *AddLabelOperation) error {
	_, _, err := cli.client.Issues.AddLabelsToIssue(ctx, op.Owner, op.Repo, op.Number, op.Labels)
	if err != nil {
		return err
	}
	return nil
}

type RemoveLabelOperation struct {
	Owner  string
	Repo   string
	Number int
	Label  string
}

func (cli *githubClient) doRemoveLabelOperation(ctx context.Context, op *RemoveLabelOperation) error {
	_, err := cli.client.Issues.RemoveLabelForIssue(ctx, op.Owner, op.Repo, op.Number, op.Label)
	if err != nil {
		return err
	}
	return nil
}

type AddAssignOperation struct {
	Owner     string
	Repo      string
	Number    int
	Assignees []string
}

func (cli *githubClient) doAddAssignOperation(ctx context.Context, op *AddAssignOperation) error {
	_, _, err := cli.client.Issues.AddAssignees(ctx, op.Owner, op.Repo, op.Number, op.Assignees)
	return err
}

type RemoveAssignOperation struct {
	Owner     string
	Repo      string
	Number    int
	Assignees []string
}

func (cli *githubClient) doRemoveAssignOperation(ctx context.Context, op *RemoveAssignOperation) error {
	_, _, err := cli.client.Issues.RemoveAssignees(ctx, op.Owner, op.Repo, op.Number, op.Assignees)
	return err
}

type CloseOperation struct {
	Owner  string
	Repo   string
	Number int
	Object *Object // can get issue or pr info from payload
}

func (cli *githubClient) doCloseOperation(ctx context.Context, op *CloseOperation) error {
	closeState := "closed"

	if _, ok := op.Object.Payload().(GetPullRequestInterface); ok {
		_, _, err := cli.client.PullRequests.Edit(ctx, op.Owner, op.Repo, op.Number, &github.PullRequest{
			State: &closeState,
		})
		return err
	}

	if _, ok := op.Object.Payload().(GetIssueInterface); ok {
		_, _, err := cli.client.Issues.Edit(ctx, op.Owner, op.Repo, op.Number, &github.IssueRequest{
			State: &closeState,
		})
		return err
	}

	return fmt.Errorf("can't get issue or pr from object")
}

type ReopenOperation struct {
	Owner  string
	Repo   string
	Number int
	Object *Object // can get issue or pr info from payload
}

func (cli *githubClient) doReopenOperation(ctx context.Context, op *ReopenOperation) error {
	openStatue := "open"

	if _, ok := op.Object.Payload().(GetPullRequestInterface); ok {
		_, _, err := cli.client.PullRequests.Edit(ctx, op.Owner, op.Repo, op.Number, &github.PullRequest{
			State: &openStatue,
		})
		return err
	}

	if _, ok := op.Object.Payload().(GetIssueInterface); ok {
		_, _, err := cli.client.Issues.Edit(ctx, op.Owner, op.Repo, op.Number, &github.IssueRequest{
			State: &openStatue,
		})
		return err
	}

	return fmt.Errorf("can't get issue or pr from object")
}
