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

type discordDriver struct {
	session *discordgo.Session
}

func (svc *Service) StartDiscord() error {
	fmt.Fprintln(os.Stderr, banner.Inline("discord"))
	svc.logger.Debug("starting discord")

	dg, err := discordgo.New("Bot " + svc.opts.DiscordToken)
	if err != nil {
		return err
	}

	// send hello message
	{
		msg := "**Hello World!**"
		if svc.opts.DevMode {
			msg += " (dev)"
		}
		_, err = dg.ChannelMessageSend(svc.opts.DiscordAdminChannel, msg)
		if err != nil {
			svc.logger.Warn("send hello message", zap.Error(err))
		}
	}

	// handlers
	{
		dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
			if m.Author.ID == s.State.User.ID {
				return
			}
			if m.Author.Bot {
				return
			}
			log.Println(godev.JSON(m))
			_, err := s.ChannelMessageSend(m.ChannelID, ">>> "+m.Content)
			if err != nil {
				svc.logger.Error("discord.ChannelMessageSend", zap.Error(err))
			}
		})
	}

	// start
	{
		err = dg.Open()
		if err != nil {
			return err
		}
		svc.discord.session = dg

		<-svc.ctx.Done()
	}
	return nil
}

func (svc *Service) CloseDiscord(error) {
	svc.logger.Debug("closing discord", zap.Bool("was-started", svc.discord.session != nil))
	if svc.discord.session != nil {
		svc.discord.session.Close()
		svc.logger.Debug("discord closed")
	}
	svc.cancel()
}
