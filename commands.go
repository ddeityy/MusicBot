package musicbot

import "github.com/bwmarrin/discordgo"

var Commands = []*discordgo.ApplicationCommand{
	// Utility
	{Name: "join", Description: "Join the voice channel you are in"},
	{Name: "leave", Description: "Leave the voice channel you are in"},

	// Queue
	{Name: "clear", Description: "Clears the queue"},
	{Name: "queue", Description: "Show the current queue"},
	{Name: "shuffle", Description: "Shuffles the queue"},

	// Music playback
	{Name: "play", Description: "Play a song from youtube"},
	{Name: "pause", Description: "Pause the current song"},
	{Name: "skip", Description: "Skip the current song"},

	// Options
	{Name: "add", Description: "Adds a song to the queue",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "URL",
				Description: "The URL of the song to add",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		}},
	{Name: "remove", Description: "Removes a song from the queue",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "Index",
				Description: "The index of the song to remove",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
			},
		}},
}
