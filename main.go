package musicbot

import (
	"fmt"
	"log"
	"os"

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

func main() {
	TOKEN = os.Getenv("TOKEN")
	GUILD = os.Getenv("GUILD")
	APP = os.Getenv("APP")
	YT = os.Getenv("YT")

	os.MkdirAll("./audio", 0655)
	os.MkdirAll("./video", 0655)

	Bot, err = discordgo.New("Bot " + TOKEN)
	if err != nil {
		panic(fmt.Errorf("error creating discord session: %s\n", err))
	}
	defer Bot.Close()

	Bot.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err := Bot.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	for _, handler := range Handlers {
		Bot.AddHandler(handler)
	}

	_, err = Bot.ApplicationCommandBulkOverwrite(APP, GUILD, Commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}
}
