package moulbotsvc

import "go.uber.org/zap"

type Service struct {
	logger *zap.Logger
	opts   Opts
}

type Opts struct {
	Logger     *zap.Logger
	ServerOpts ServerOpts
}

func New(opts Opts) Service {
	opts.applyDefaults()
	return Service{
		logger: opts.Logger,
		opts:   opts,
	}
}

func (s *Service) Close() {}

func (opts *Opts) applyDefaults() {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
}
