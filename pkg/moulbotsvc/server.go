package moulbotsvc

import "go.uber.org/zap"

type ServerOpts struct{}

func (s *Service) StartServer() error {
	opts := s.opts.ServerOpts
	opts.applyDefaults()
	s.logger.Info("start server", zap.Any("opts", opts))
	return nil
}

func (s *Service) CloseServer(error) {}

func (opts *ServerOpts) applyDefaults() {}
