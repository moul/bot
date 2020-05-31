package moulbot

import (
	"context"

	"moul.io/moul-bot/pkg/moulbotpb"
)

func (svc *Service) Ping(context.Context, *moulbotpb.Ping_Request) (*moulbotpb.Ping_Response, error) {
	return &moulbotpb.Ping_Response{}, nil
}
