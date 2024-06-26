package main

import (
	"flag"
	"os"
)

var (
	TOKEN string
	GUILD string
	APP   string
	YT    string
)

func init() {
	os.MkdirAll("./audio", 0755)

	tokenFlag := flag.String("token", "", "Your Discord bot token")
	guildFlag := flag.String("guild", "", "Guild ID where the bot operates")
	appFlag := flag.String("app", "", "Application ID for Discord bot")
	ytFlag := flag.String("yt", "", "YouTube API Key")

	flag.Parse()

	TOKEN = *tokenFlag
	GUILD = *guildFlag
	APP = *appFlag
	YT = *ytFlag
}
