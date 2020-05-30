package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/oklog/run"
	ff "github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"moul.io/banner"
	"moul.io/godev"
	"moul.io/moul-bot/pkg/moulbotsvc"
)

func main() {
	if err := cli(os.Args); err != nil {
		log.Fatalf("error: %v", err)
		os.Exit(1)
	}
}

var (
	// FIXME: add zap
	discordToken        string
	discordAdminChannel string
)

func cli(args []string) error {
	rootFlags := flag.NewFlagSet("root", flag.ExitOnError)
	rootFlags.StringVar(&discordToken, "discord-token", "", "Discord Bot Token")
	rootFlags.StringVar(&discordAdminChannel, "discord-admin-channel", "", "Discord channel ID for admin messages")
	root := &ffcli.Command{
		FlagSet: rootFlags,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("MOULBOT"),
			ff.WithConfigFile("config.txt"),
			ff.WithConfigFileParser(ff.PlainParser),
		},
		Subcommands: []*ffcli.Command{
			{Name: "discord-bot", Exec: discordBotCmd},
			{Name: "api-server", Exec: apiServerCmd},
			// FIXME: make a mode that starts a unique bot with multiple interfaces (disocord, http, grpc, ssh, etc)
		},
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}
	return root.ParseAndRun(context.Background(), args[1:])
}

func apiServerCmd(ctx context.Context, _ []string) error {
	opts := moulbotsvc.Opts{}
	svc := moulbotsvc.New(opts)
	defer svc.Close()
	gr := run.Group{}
	gr.Add(run.SignalHandler(ctx, syscall.SIGKILL, syscall.SIGTERM))
	gr.Add(svc.StartServer, svc.CloseServer)
	return gr.Run()
}

func discordBotCmd(ctx context.Context, _ []string) error {
	fmt.Fprintln(os.Stderr, banner.Inline("moul-bot - discord"))
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		return err
	}
	_, err = dg.ChannelMessageSend(discordAdminChannel, fmt.Sprintf("**Hello World!**"))
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
			panic(err)
		}
	})
	err = dg.Open()
	if err != nil {
		return err
	}
	defer dg.Close()
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	var g run.Group
	g.Add(run.SignalHandler(ctx, os.Interrupt))
	return g.Run()
}
