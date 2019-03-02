package notify

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatedier/freebot/pkg/log"
	"github.com/fatedier/freebot/pkg/util"
)

const (
	WaitReviewPullRequst = "wait_review_pull_request"
)

var (
	crontabList = []string{WaitReviewPullRequst}
)

func (p *NotifyPlugin) RunTask() {
	if len(p.extra.Crontab) == 0 {
		return
	}

	for _, ctype := range crontabList {
		crontab, ok := p.extra.Crontab[ctype]
		if !ok || crontab.Disable {
			continue
		}
		switch ctype {
		case WaitReviewPullRequst:
			p.cron.AddFunc(crontab.Job, p.runWaitReviewJob)
		}
	}
	p.cron.Start() // will start with a goroutine
	return
}

func (p *NotifyPlugin) runWaitReviewJob() {
	crontab := p.extra.Crontab[WaitReviewPullRequst]
	prs, err := p.cli.ListPullRequestsByState(p.cliCtx, p.GetOwner(), p.GetRepo(), "open")
	if err != nil {
		log.Warn("list pull request error: %v", err)
		return
	}

	userMsg := make(map[string][]string, 0)
	for _, pr := range prs {
		for _, ru := range pr.RequestedReviewers {
			if util.StringContains(crontab.SendToUsers, ru) {
				userMsg[ru] = append(userMsg[ru], fmt.Sprintf("%s [%s] [%s]", pr.HTMLURL, pr.User, pr.Title))
			}
		}
	}

	for uk, uv := range userMsg {
		content := fmt.Sprintf("[%s/%s] Pull Requests Wait To Review\n", p.GetOwner(), p.GetRepo())
		if notifyOption, ok := p.extra.UserNotifyConfs[uk]; ok {
			content += strings.Join(uv, "\n")
			err = p.notifier.Send(context.Background(), notifyOption, content)
			if err != nil {
				log.Warn("run crontab job error: %v", err)
				continue
			}
			log.Info("send request review pull request message to [%v] successfully, message: [%v]", uk, content)
		}
	}
}
