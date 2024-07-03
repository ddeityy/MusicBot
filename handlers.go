package main

import (
	"fmt"
	"log"
	"time"

	URL "net/url"

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

var titleChan = make(chan string, 0)
var idChan = make(chan string, 0)

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
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintln("Joined!"),
		},
	})

	log.Println("Joined voice channel")
}

func handleLeave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := VC.Speaking(false)
	if err != nil {
		log.Println("Error disabling voice: ", err)
		return
	}

	if err = VC.Disconnect(); err != nil {
		log.Println("Error leaving voice channel: ", err)
		return
	}

	inVC = false

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintln("Left!"),
		},
	})

	log.Println("Left voice channel")
}

func handleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}})

	response := &discordgo.WebhookEdit{}

	switch i.ApplicationCommandData().Options[0].Name {
	case "file":
		attachmentID := i.ApplicationCommandData().Options[0].Value.(string)
		attachmentUrl := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL
		song, err := downloadAttachment(attachmentUrl)
		if err != nil {
			log.Println(err)
			errString := err.Error()
			response.Content = &errString
			s.InteractionResponseEdit(i.Interaction, response)
			return
		}
		Queue.mu.Lock()
		Queue.songs = append(Queue.songs, song)
		Queue.mu.Unlock()

		success := fmt.Sprintf("Successfully added: %s", song.title)
		log.Println(success)
		response.Content = &success
		s.InteractionResponseEdit(i.Interaction, response)

		if !inVC {
			handleJoin(s, i)
		}

		if !isPlaying {
			go Queue.PlaySong()
		}

		return
	case "url":
		url := i.ApplicationCommandData().Options[0].StringValue()
		u, err := URL.Parse(url)
		if err != nil {
			log.Println(err)
			errString := err.Error()
			response.Content = &errString
			s.InteractionResponseEdit(i.Interaction, response)
			return
		}

		if !IsYouTubeURL(u) {
			log.Println(err)
			errString := "invalid YT link"
			response.Content = &errString
			s.InteractionResponseEdit(i.Interaction, response)
			return
		}

		ids, err := GetSongID(*u)
		if err != nil {
			log.Println(err)
			errString := err.Error()
			response.Content = &errString
			s.InteractionResponseEdit(i.Interaction, response)
			return
		}

		if len(ids) == 1 {
			var title string
			if title, err = Queue.AddSong(*u, ids[0]); err != nil {
				log.Println(err)
				errString := err.Error()
				response.Content = &errString
				s.InteractionResponseEdit(i.Interaction, response)
				return
			}
			success := fmt.Sprintf("Successfully added: %s", title)
			log.Println(success)
			response.Content = &success
			s.InteractionResponseEdit(i.Interaction, response)

			if !inVC {
				handleJoin(s, i)
			}

			if !isPlaying {
				go Queue.PlaySong()
			}

			return
		}

		for _, id := range ids {
			tempId := id
			time.Sleep(200 * time.Millisecond)
			go func() error {
				if _, err := Queue.AddSong(*u, tempId); err != nil {
					log.Println(err)
					errString := err.Error()
					response.Content = &errString
					s.InteractionResponseEdit(i.Interaction, response)
					return err
				}
				return nil
			}()
		}

		success := fmt.Sprintf("Successfully added: %d songs", len(ids))
		log.Println(success)
		response.Content = &success
		s.InteractionResponseEdit(i.Interaction, response)

		if !inVC {
			handleJoin(s, i)
		}

		if !isPlaying {
			go Queue.PlaySong()
		}
	}
}

func handleRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	index := int(i.ApplicationCommandData().Options[0].IntValue())

	title, err := Queue.RemoveSong(index)
	if err != nil {
		log.Println("Error removing song from queue: ", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("Error removing song from queue: %s", err),
			},
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Successfully removed: %s", title),
		},
	})

	log.Println("Successfully removed: ", title)
}

func handleQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: Queue.FormatQueue(),
		},
	})
}

func handleShuffle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	Queue.mu.Lock()
	Queue.Shuffle()
	Queue.mu.Unlock()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Successfully shuffled! New queue:\n%s", Queue.FormatQueue()),
		},
	})
}

func handleClear(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	Queue.mu.Lock()
	Queue.Empty()
	Queue.mu.Unlock()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
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
		return
	}

	if Queue.IsEmpty() {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Queue is empty, use /add"),
			},
		})
		return
	}

	Queue.mu.Lock()
	song := Queue.GetCurrentSong()
	Queue.mu.Unlock()

	if !inVC {
		handleJoin(s, i)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Playing %s", song.title),
		},
	})

	go Queue.PlaySong()
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
		return
	}

	if !isPaused {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Pausing"),
			},
		})
		Queue.PausePlayback()
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Resuming"),
		},
	})
	Queue.ResumePlayback()
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
		return
	}

	if Queue.IsEmpty() {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Queue is empty, use /add"),
			},
		})
		return
	}

	Queue.mu.Lock()
	Queue.SkipSong()
	Queue.mu.Unlock()

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Skipped"),
		},
	})
}
