package client

import (
	"context"

	"github.com/google/go-github/github"
)

func (cli *githubClient) CheckMergeable(ctx context.Context, owner, repo string, number int) (bool, error) {
	pr, _, err := cli.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, err
	}

	if pr == nil {
		return false, nil
	}

	return pr.GetMergeable(), nil
}

func (cli *githubClient) ListPullRequestBySHA(ctx context.Context, owner, repo string, sha string) (prs []PullRequest, err error) {
	githubPRs, _, err := cli.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{})
	if err != nil {
		return nil, err
	}

	prs = make([]PullRequest, 0, len(githubPRs))
	for _, githubPR := range githubPRs {
		if githubPR.GetHead().GetSHA() != sha {
			continue
		}

		pr := PullRequest{
			Number:  githubPR.GetNumber(),
			State:   githubPR.GetState(),
			Title:   githubPR.GetTitle(),
			Body:    githubPR.GetBody(),
			User:    githubPR.GetUser().GetLogin(),
			Labels:  make([]string, 0),
			HTMLURL: githubPR.GetHTMLURL(),
		}
		for _, l := range githubPR.Labels {
			pr.Labels = append(pr.Labels, l.GetName())
		}
		prs = append(prs, pr)
	}
	return
}

func (cli *githubClient) ListFilesByPullRequest(ctx context.Context, owner, repo string, number int) (files []string, err error) {
	files = make([]string, 0)
	commitFiles, _, err := cli.client.PullRequests.ListFiles(ctx, owner, repo, number, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, commitFile := range commitFiles {
		files = append(files, commitFile.GetFilename())
	}
	return
}

// operations
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
