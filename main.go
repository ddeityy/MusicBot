package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
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

var (
	tokenFlag = flag.String("token", "", "Your Discord bot token")
	guildFlag = flag.String("guild", "", "Guild ID where the bot operates")
	appFlag   = flag.String("app", "", "Application ID for Discord bot")
	ytFlag    = flag.String("yt", "", "YouTube API Key")
)

func main() {
	flag.Parse()

	TOKEN = *tokenFlag
	GUILD = *guildFlag
	APP = *appFlag
	YT = *ytFlag

	os.MkdirAll("./audio", 0755)
	os.MkdirAll("./video", 0755)

	Bot, err = discordgo.New("Bot " + TOKEN)
	if err != nil {
		log.Fatalf("error creating discord session: %s\n", err)
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
