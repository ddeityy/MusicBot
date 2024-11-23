package main

import (
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

func main() {
	var err error

	lg := NewLogger()

	session, err := discordgo.New("Bot " + TOKEN)
	if err != nil {
		lg.Error("error creating discord session: ", err)
		os.Exit(1)
	}

	ch := NewCommandHandler(lg)

	var handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"join":    ch.handleJoin,
		"leave":   ch.handleLeave,
		"add":     ch.handleAdd,
		"remove":  ch.handleRemove,
		"pause":   ch.handlePauseResume,
		"queue":   ch.handleQueue,
		"shuffle": ch.handleShuffle,
		"skip":    ch.handleSkip,
		"clear":   ch.handleClear,
	}

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		lg.Info("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	_, err = session.ApplicationCommandBulkOverwrite(APP, GUILD, Commands)
	if err != nil {
		lg.Error("Could not register commands: %s", err)
		os.Exit(1)
	}

	err = session.Open()
	if err != nil {
		lg.Error("Could not open session: %s", err)
		os.Exit(1)
	}

	go RunServer()

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		lg.Error("could not close session gracefully: %s", err)
		os.Exit(1)
	}
}
