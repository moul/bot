package moulbot

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"moul.io/banner"
)

func (svc *Service) StartServer() error {
	fmt.Fprintln(os.Stderr, banner.Inline("server"))
	svc.logger.Info("start server", zap.String("bind", svc.opts.ServerBind))

	<-svc.ctx.Done()
	return nil
}

func (svc *Service) CloseServer(error) {
	svc.logger.Debug("closing server")
	svc.cancel()
}
