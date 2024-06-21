package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	TOKEN string
	GUILD string
	APP   string
	YT    string

	err error

	Q   *Queue
	Bot *discordgo.Session
)

func main() {
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	TOKEN = os.Getenv("TOKEN")
	GUILD = os.Getenv("GUILD")
	APP = os.Getenv("APP")
	YT = os.Getenv("YT")

	os.MkdirAll("./audio", 0755)
	os.MkdirAll("./video", 0755)

	Bot, err = discordgo.New("Bot " + TOKEN)
	if err != nil {
		panic(fmt.Errorf("error creating discord session: %s\n", err))
	}
	defer Bot.Close()

	Q = NewQueue()

	Bot.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	Bot.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := Handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	_, err = Bot.ApplicationCommandBulkOverwrite(APP, GUILD, Commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	err = Bot.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = Bot.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}
