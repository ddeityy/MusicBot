package main

import (
	"github.com/bwmarrin/discordgo"
)

func (ch *CommandHandler) Error(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	response := &discordgo.WebhookEdit{}
	e := err.Error()
	response.Content = &e
	s.InteractionResponseEdit(i.Interaction, response)
}

func (ch *CommandHandler) Success(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: msg,
		},
	})
}

func (ch *CommandHandler) WaitSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	response := &discordgo.WebhookEdit{}
	response.Content = &msg
	s.InteractionResponseEdit(i.Interaction, response)
}

func (ch *CommandHandler) Wait(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})
}
