package moulbot

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v32/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"moul.io/banner"
)

func (svc *Service) StartGitHub() error {
	fmt.Fprintln(os.Stderr, banner.Inline("github"))
	svc.logger.Debug(
		"starting github",
		zap.Bool("moul", svc.opts.GitHubMoulToken != ""),
		zap.Bool("moul-bot", svc.opts.GitHubMoulBotToken != ""),
	)

	svc.githubMoulClient = githubClient(svc.opts.GitHubMoulToken)
	svc.githubMoulBotClient = githubClient(svc.opts.GitHubMoulBotToken)

	// FIXME: print my info on init
	// FIXME: check activity
	// FIXME: print quota

	<-svc.ctx.Done()
	return nil
}

func (svc *Service) CloseGitHub(error) {
	svc.logger.Debug("closing github")
	svc.cancel()
}

func githubClient(token string) *github.Client {
	if token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		return github.NewClient(tc)
	}

	return github.NewClient(nil)
}
