package moulbot

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v32/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"moul.io/banner"
	"moul.io/godev"
)

type githubDriver struct {
	clients map[string]*github.Client
}

func (svc *Service) StartGitHub() error {
	fmt.Fprintln(os.Stderr, banner.Inline("github"))
	svc.logger.Debug(
		"starting github",
		zap.Bool("moul", svc.opts.GitHubMoulToken != ""),
		zap.Bool("moul-bot", svc.opts.GitHubMoulbotToken != ""),
	)

	svc.github.clients = make(map[string]*github.Client)
	svc.github.clients["moul"] = githubClient(svc.opts.GitHubMoulToken)
	svc.github.clients["moul-bot"] = githubClient(svc.opts.GitHubMoulbotToken)

	// get myself
	for key := range svc.github.clients {
		err := svc.githubMyself(key)
		if err != nil {
			return err
		}
	}

	go svc.githubRoutine()

	<-svc.ctx.Done()
	return nil
}

func (svc *Service) githubRoutine() {
	alltimeMostRecent := time.Time{}
	for {
		// check activity
		events, _, err := svc.github.clients["moul"].Activity.ListEventsPerformedByUser(svc.ctx, "moul", false, nil)
		if err != nil {
			svc.logger.Error("ListEventsPerformedByUser", zap.Error(err))
			continue
		}

		batchMostRecent := time.Time{}
		for _, event := range events {
			if !event.GetCreatedAt().After(alltimeMostRecent) {
				continue
			}
			if event.GetCreatedAt().After(batchMostRecent) {
				batchMostRecent = event.GetCreatedAt()
			}
			eventLogger := svc.logger.With(zap.String("type", event.GetType()))
			if event.GetPublic() {
				eventLogger = eventLogger.With(zap.String("visibility", "public"))
			} else {
				eventLogger = eventLogger.With(zap.String("visibility", "private"))
			}
			if event.Repo != nil {
				eventLogger = eventLogger.With(zap.String("repo", event.GetRepo().GetName()))
			} else {
				eventLogger = eventLogger.With(zap.String("repo", "n/a"))
			}
			if event.Actor != nil {
				eventLogger = eventLogger.With(zap.String("actor", event.GetActor().GetLogin()))
			} else {
				eventLogger = eventLogger.With(zap.String("actor", "n/a"))
			}
			if event.Org != nil {
				eventLogger = eventLogger.With(zap.String("org", event.GetOrg().GetLogin()))
			} else {
				eventLogger = eventLogger.With(zap.String("org", "n/a"))
			}
			switch *event.Type {
			case "PushEvent", "DeleteEvent":
				// skipped
			case "PullRequestEvent", "IssueCommentEvent", "IssuesEvent", "MemberEvent":
				// light
				eventLogger.Debug("event")
			case "CreateEvent":
				payload, err := event.ParsePayload()
				if err != nil {
					eventLogger.Error("event.ParsePayload", zap.Error(err))
				} else {
					typed, valid := payload.(*github.CreateEvent)
					if !valid {
						eventLogger.Error("invalid payload")
						continue
					}
					if typed.GetRefType() == "repository" {
						eventLogger.Info("new repo", zap.String("name", event.GetRepo().GetName()))
						parts := strings.Split(event.GetRepo().GetName(), "/")

						// moul invites moul-bot
						var invite *github.CollaboratorInvitation
						{
							svc.logger.Debug("trying to send an invite to moul-bot")
							var err error
							invite, _, err = svc.github.clients["moul"].Repositories.AddCollaborator(svc.ctx, parts[0], parts[1], "moul-bot", nil)
							if err != nil {
								svc.logger.Error("AddCollaborator", zap.Error(err))
								continue
							}
						}
						// moul-bot accepts the invitation
						{
							if invite.ID != nil {
								svc.logger.Debug("accepting invite")
								_, err = svc.github.clients["moul-bot"].Users.AcceptInvitation(svc.ctx, invite.GetID())
								if err != nil {
									svc.logger.Error("AcceptInvitation", zap.Error(err))
								}
							}
						}
						// moul-bot stars the repository
						{
							_, err := svc.github.clients["moul-bot"].Activity.Star(svc.ctx, parts[0], parts[1])
							if err != nil {
								svc.logger.Error("Star", zap.Error(err))
							}
						}
					} else {
						eventLogger.Warn("unknown CreateEvent RefType", zap.Any("payload", payload))
					}
				}
			default:
				eventLogger.Warn("unknown Event type")
				fmt.Println("unknown type", godev.PrettyJSON(event))
			}
		}
		if alltimeMostRecent.Before(batchMostRecent) {
			alltimeMostRecent = batchMostRecent
		}
		for key := range svc.github.clients {
			svc.githubRateLimit(key)
		}
		select {
		case <-time.After(10 * time.Second):
		case <-svc.ctx.Done():
			svc.logger.Debug("context done")
			return
		}
	}
}

func (svc *Service) githubLogger(client string) *zap.Logger {
	return svc.logger.With(zap.String("client", client))
}

func (svc *Service) githubRateLimit(client string) {
	logger := svc.githubLogger(client)
	rate, _, err := svc.github.clients[client].RateLimits(svc.ctx)
	if err != nil {
		logger.Warn("ratelimit error", zap.Error(err))
		return
	}
	logger.Debug(
		"ratelimit",
		zap.Int("remaining", rate.GetCore().Remaining),
		zap.Int("limit", rate.GetCore().Limit),
	)
}

func (svc *Service) githubMyself(client string) error {
	logger := svc.githubLogger(client)
	user, _, err := svc.github.clients[client].Users.Get(svc.ctx, "")
	if err != nil {
		return err
	}
	logger.Debug(
		"myself",
		zap.String("name", user.GetLogin()),
		zap.String("name", user.GetName()),
		zap.Int("followers", user.GetFollowers()),
		zap.Int("following", user.GetFollowing()),
	)
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
