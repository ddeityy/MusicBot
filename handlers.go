package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

var VC *discordgo.VoiceConnection
var inVC bool

var Handlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"join":    handleJoin,
	"leave":   handleLeave,
	"add":     handleAdd,
	"remove":  handleRemove,
	"pause":   handlePauseResume,
	"play":    handlePlay,
	"queue":   handleQueue,
	"shuffle": handleShuffle,
	"skip":    handleSkip,
	"clear":   handleClear,
}

func handleJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	c, err := s.State.Channel(i.ChannelID)
	if err != nil {
		log.Println("Error getting channel state: ", err)
		return
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		log.Println("Error getting guild state: ", err)
		return
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID == i.Member.User.ID {
			VC, err = s.ChannelVoiceJoin(GUILD, vs.ChannelID, false, false)
			if err != nil {
				log.Println("Error joining voice channel: ", err)
				return
			}
			inVC = true
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintln("Joined!"),
		},
	})

}

func handleLeave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err = VC.Speaking(false)
	if err != nil {
		log.Println(err)
	}
	if err = VC.Disconnect(); err != nil {
		log.Println("Error leaving voice channel: ", err)
		return
	}
	inVC = false
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintln("Left!"),
		},
	})
}

func handleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	url := i.ApplicationCommandData().Options[0].StringValue()
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{}})

	response := &discordgo.WebhookEdit{}

	if !inVC {
		handleJoin(s, i)
	}

	Q.mu.Lock()
	title, err := Q.AddSong(url)
	if err != nil {
		errString := err.Error()
		response.Content = &errString
		s.InteractionResponseEdit(i.Interaction, response)
		return
	}
	Q.mu.Unlock()

	success := fmt.Sprintf("Successfully added: %s", title)
	response.Content = &success
	s.InteractionResponseEdit(i.Interaction, response)

	if !isPlaying {
		if len(Q.songs) == 1 {
			go Q.PlaySong()
		}
	}
}

func handleRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	index := int(i.ApplicationCommandData().Options[0].IntValue())

	title, err := Q.RemoveSong(index)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Error removing song from queue: %s", err),
			},
		})
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully removed: %s", title),
		},
	})
}

func handleQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: Q.FormatQueue(),
		},
	})
}

func handleShuffle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	Q.mu.Lock()
	Q.Shuffle()
	Q.mu.Unlock()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Successfully shuffled! New queue:\n%s", Q.FormatQueue()),
		},
	})
}

func handleClear(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	Q.mu.Lock()
	Q.Empty()
	Q.mu.Unlock()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintln("Successfully cleared!"),
		},
	})
}

func handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if isPlaying {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Already playing"),
			},
		})
	}

	if Q.IsEmpty() {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Queue is empty, use /add"),
			},
		})
		return
	}

	Q.mu.Lock()
	song := Q.GetCurrentSong()
	Q.mu.Unlock()

	if !inVC {
		handleJoin(s, i)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Playing %s", song.Title),
		},
	})

	go Q.PlaySong()
}

func handlePauseResume(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if !inVC {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Not in voice chat, use /join or /play"),
			},
		})
	}

	if isPaused {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Resuming"),
			},
		})
		isPlaying = true
		Q.ResumePlayback()
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Pausing"),
			},
		})
		isPlaying = false
		Q.PausePlayback()
	}
}

func handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	if !inVC {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Not in voice chat, use /join"),
			},
		})
	}

	if Q.IsEmpty() {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Queue is empty, use /add"),
			},
		})
	}

	Q.mu.Lock()
	Q.SkipSong()
	Q.mu.Unlock()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Skipped"),
		},
	})
}
