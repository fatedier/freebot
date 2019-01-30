package client

import (
	"context"

	"github.com/google/go-github/github"
)

type RequestReviewsOperation struct {
	Owner     string
	Repo      string
	Number    int
	Reviewers []string
}

func (cli *githubClient) doRequestReviewsOperation(ctx context.Context, op *RequestReviewsOperation) error {
	_, _, err := cli.client.PullRequests.RequestReviewers(ctx, op.Owner, op.Repo, op.Number, github.ReviewersRequest{
		Reviewers: op.Reviewers,
	})
	return err
}

type RequestReviewsCancelOperation struct {
	Owner           string
	Repo            string
	Number          int
	CancelReviewers []string
}

func (cli *githubClient) doRequestReviewsCancelOperation(ctx context.Context, op *RequestReviewsCancelOperation) error {
	_, err := cli.client.PullRequests.RemoveReviewers(ctx, op.Owner, op.Repo, op.Number, github.ReviewersRequest{
		Reviewers: op.CancelReviewers,
	})
	return err
}

type MergeOperation struct {
	Owner  string
	Repo   string
	Number int
}

func (cli *githubClient) doMergeOperation(ctx context.Context, op *MergeOperation) error {
	_, _, err := cli.client.PullRequests.Merge(ctx, op.Owner, op.Repo, op.Number, "auto merged by freebot", nil)
	return err
}
