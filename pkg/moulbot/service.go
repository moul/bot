package moulbot

import (
	"context"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"moul.io/banner"
)

type Service struct {
	logger *zap.Logger
	opts   Opts
	ctx    context.Context
	cancel func()

	discord *discordgo.Session
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
	svc.logger.Info("start service", zap.Any("opts", opts.Filtered()))
	return svc
}

func (svc *Service) Close() {
	svc.logger.Debug("closing service")
	svc.cancel()
}
