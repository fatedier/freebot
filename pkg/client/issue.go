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
	labels, _, err := cli.client.Issues.ListLabels(ctx, op.Owner, op.Repo, &github.ListOptions{})
	if err != nil {
		return err
	}
	labelsMap := make(map[string]struct{})
	for _, l := range labels {
		labelsMap[l.GetName()] = struct{}{}
	}

	oldLabels, _, err := cli.client.Issues.ListLabelsByIssue(ctx, op.Owner, op.Repo, op.Number, &github.ListOptions{})
	if err != nil {
		return err
	}

	newLabels := make([]string, 0)
	for _, name := range op.Labels {
		if _, ok := labelsMap[name]; ok {
			newLabels = append(newLabels, name)
		}
	}
	if len(newLabels) == 0 {
		return fmt.Errorf("no replace lables")
	}

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
