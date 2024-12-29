package main

import (
	"github.com/bwmarrin/discordgo"
)

func (ch *CommandHandler) Error(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	response := &discordgo.WebhookEdit{}
	e := err.Error()
	response.Content = &e
	_, err = s.InteractionResponseEdit(i.Interaction, response)
	if err != nil {
		ch.lg.Error("Error editing interaction: ", err)
	}
}

func (ch *CommandHandler) Success(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: msg,
		},
	})
	if err != nil {
		ch.lg.Error("Error responding to interaction: ", err)
	}
}

func (ch *CommandHandler) WaitSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	response := &discordgo.WebhookEdit{}
	response.Content = &msg
	_, err := s.InteractionResponseEdit(i.Interaction, response)
	if err != nil {
		ch.lg.Error("Error editing interaction: ", err)
	}
}

func (ch *CommandHandler) Wait(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}},
	)
	if err != nil {
		ch.lg.Error("Error responding to interaction: ", err)
	}
}
