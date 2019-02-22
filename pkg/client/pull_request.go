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
	prs = make([]PullRequest, 0)
	step := 50
	page := 1
	for {
		githubPRs, _, err := cli.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 50,
			},
		})
		if err != nil {
			return nil, err
		}

		// no more pull requests
		if len(githubPRs) == 0 {
			break
		}
		page++

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

		// no more pull requests
		if len(githubPRs) < step {
			break
		}
	}
	return
}

func (cli *githubClient) ListFilesByPullRequest(ctx context.Context, owner, repo string, number int) (files []string, err error) {
	files = make([]string, 0)
	step := 200
	page := 1
	for {
		commitFiles, _, err := cli.client.PullRequests.ListFiles(ctx, owner, repo, number, &github.ListOptions{
			Page:    page,
			PerPage: step,
		})
		if err != nil {
			return nil, err
		}

		// no more files
		if len(commitFiles) == 0 {
			break
		}
		page++

		for _, commitFile := range commitFiles {
			files = append(files, commitFile.GetFilename())
		}

		// no more files
		if len(commitFiles) < step {
			break
		}
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
