package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

var Session *discordgo.Session
var Queue *SongQueue

func main() {
	var err error
	Session, err = discordgo.New("Bot " + TOKEN)
	if err != nil {
		log.Fatalf("error creating discord session: %s\n", err)
	}
	Queue = NewSongQueue()

	defer Session.Close()

	Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := Handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	_, err = Session.ApplicationCommandBulkOverwrite(APP, GUILD, Commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	err = Session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = Session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}
