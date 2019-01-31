package client

import (
	"context"
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
