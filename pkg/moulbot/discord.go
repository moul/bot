package moulbot

import (
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"moul.io/banner"
	"moul.io/godev"
)

func (svc *Service) StartDiscord() error {
	fmt.Fprintln(os.Stderr, banner.Inline("discord"))
	svc.logger.Info("starting discord")

	dg, err := discordgo.New("Bot " + svc.opts.DiscordToken)
	if err != nil {
		return err
	}
	_, err = dg.ChannelMessageSend(svc.opts.DiscordAdminChannel, fmt.Sprintf("**Hello World!**"))
	if err != nil {
		return err
	}
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		log.Println(godev.JSON(m))
		_, err := s.ChannelMessageSend(m.ChannelID, ">>> "+m.Content)
		if err != nil {
			svc.logger.Error("discord.ChannelMessageSend", zap.Error(err))
		}
	})
	err = dg.Open()
	if err != nil {
		return err
	}
	svc.discord = dg

	<-svc.ctx.Done()
	return nil
}

func (svc *Service) CloseDiscord(error) {
	svc.logger.Debug("closing discord", zap.Bool("was-started", svc.discord != nil))
	if svc.discord != nil {
		svc.discord.Close()
		svc.logger.Debug("discord closed")
		svc.cancel()
	}
}
