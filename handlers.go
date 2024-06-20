package musicbot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

var VC *discordgo.VoiceConnection

var Handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"join":  handleJoin,
	"leave": handleLeave,
	"add":   handleAdd,
}

func handleJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	VC, err = s.ChannelVoiceJoin(i.GuildID, i.ChannelID, false, false)
	if err != nil {
		log.Println("Error joining voice channel: ", err)
	}
}

func handleLeave(_ *discordgo.Session, _ *discordgo.InteractionCreate) {
	if err = VC.Disconnect(); err != nil {
		log.Println("Error leaving voice channel: ", err)
	}
}

func handleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	url := i.ApplicationCommandData().Options[0]

	if err = Q.AddSong(url.StringValue()); err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error adding song to queue: %s", err),
			},
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully added: %s", Q.GetLastSong().Title),
		},
	})
}
