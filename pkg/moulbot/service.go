package moulbot

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/google/go-github/v32/github"
	"go.uber.org/zap"
	"moul.io/banner"
)

type Service struct {
	logger *zap.Logger
	opts   Opts
	ctx    context.Context
	cancel func()

	/// discord

	discord *discordgo.Session

	/// server

	serverListener net.Listener

	/// github

	githubMoulClient    *github.Client
	githubMoulBotClient *github.Client
}

func New(opts Opts) Service {
	opts.applyDefaults()
	fmt.Fprintln(os.Stderr, banner.Inline("moul-bot"))
	ctx, cancel := context.WithCancel(opts.Context)
	svc := Service{
		logger: opts.Logger,
		opts:   opts,
		ctx:    ctx,
		cancel: cancel,
	}
	svc.logger.Info("service initialized", zap.Bool("dev-mode", opts.DevMode))
	return svc
}

func (svc *Service) Close() {
	svc.logger.Debug("closing service")
	svc.cancel()
	fmt.Fprintln(os.Stderr, banner.Inline("kthxbie"))
}
