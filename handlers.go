package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var titleChan = make(chan string, 0)
var idChan = make(chan string, 0)

func (ch *CommandHandler) handleJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleJoin: "

	c, err := s.State.Channel(i.ChannelID)
	if err != nil {
		ch.lg.Error(op+"Error getting channel state: ", err)
		ch.Error(s, i, fmt.Errorf("Error getting channel state: %w", err))
		return
	}

	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		ch.lg.Error(op+"Error getting guild state: ", err)
		ch.Error(s, i, fmt.Errorf("Error getting guild state: %w", err))
		return
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID == i.Member.User.ID {
			ch.voiceConn, err = s.ChannelVoiceJoin(GUILD, vs.ChannelID, false, false)
			if err != nil {
				ch.lg.Error(op+"Error joining voice channel: ", err)
				ch.Error(s, i, fmt.Errorf("Error joining voice channel: %w", err))
				return
			}
		}
	}

	ch.Success(s, i, "Joined")

	ch.lg.Info("Joined voice channel")
}

func (ch *CommandHandler) handleLeave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleLeave: "

	err := ch.voiceConn.Speaking(false)
	if err != nil {
		ch.lg.Error(op+"Error disabling voice: ", err)
		ch.Error(s, i, fmt.Errorf("Error disabling voice: %w", err))
		return
	}

	ch.isSpeaking = false

	if err = ch.voiceConn.Disconnect(); err != nil {
		ch.lg.Error(op+"Error leaving voice channel: ", err)
		ch.Error(s, i, fmt.Errorf("Error leaving voice channel: %w", err))
		return
	}

	ch.Success(s, i, "Left")

	ch.lg.Info("Left voice channel")
}

func (ch *CommandHandler) handleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleAdd: "

	if i.Type != discordgo.InteractionApplicationCommand {
		ch.lg.Error(op+"Invalid interaction type: ", fmt.Errorf("%v", i.Type))
		ch.Error(s, i, fmt.Errorf("Invalid interaction type: %s", i.Type.String()))
		return
	}

	if len(i.ApplicationCommandData().Options) == 0 {
		ch.lg.Error(op + "No options provided")
		ch.Error(s, i, fmt.Errorf("no options provided"))
		return
	}

	switch i.ApplicationCommandData().Options[0].Name {
	case "file":
		err := ch.HandleFileAttachment(s, i)
		if err != nil {
			ch.lg.Error(op+"Error downloading attachment: ", err)
			ch.Error(s, i, fmt.Errorf("Error downloading attachment: %w", err))
			return
		}
	case "url":
		err := ch.HandleYouTubeURL(s, i)
		if err != nil {
			ch.lg.Error(op+"Error adding song: ", err)
			ch.Error(s, i, fmt.Errorf("Error adding song: %w", err))
			return
		}
	}

	ch.Success(s, i, "Added to queue")

	if ch.voiceConn == nil {
		ch.handleJoin(s, i)
	}

	go ch.PlaySong()
}

func (ch *CommandHandler) handleRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleRemove: "

	if i.Type != discordgo.InteractionApplicationCommand {
		ch.lg.Error(op+"Invalid interaction type: ", fmt.Errorf("%v", i.Type))
		ch.Error(s, i, fmt.Errorf("Invalid interaction type: %s", i.Type.String()))
		return
	}

	index := int(i.ApplicationCommandData().Options[0].IntValue())

	title, err := ch.RemoveSong(index)
	if err != nil {
		ch.lg.Error(op+"Error removing song from queue: ", err)
		ch.Error(s, i, fmt.Errorf("Error removing song from queue: %w", err))
	}

	ch.Success(s, i, "Removed from queue")
	ch.lg.Info("Successfully removed: %s", title)
}

func (ch *CommandHandler) handleQueue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleQueue: "

	if i.Type != discordgo.InteractionApplicationCommand {
		ch.lg.Error(op+"Invalid interaction type: ", fmt.Errorf("%v", i.Type))
		ch.Error(s, i, fmt.Errorf("Invalid interaction type: %s", i.Type.String()))
		return
	}

	ch.Success(s, i, ch.GetFormattedQueue())
	ch.lg.Info("Successfully sent queue")
}

func (ch *CommandHandler) handleShuffle(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleShuffle: "

	if i.Type != discordgo.InteractionApplicationCommand {
		ch.lg.Error(op+"Invalid interaction type: ", fmt.Errorf("%v", i.Type))
		ch.Error(s, i, fmt.Errorf("Invalid interaction type: %s", i.Type.String()))
		return
	}

	ch.mu.Lock()
	ch.Shuffle()
	ch.mu.Unlock()

	ch.Success(s, i, "Shuffled")
	ch.lg.Info("Successfully shuffled queue")
}

func (ch *CommandHandler) handleClear(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleClear: "

	if i.Type != discordgo.InteractionApplicationCommand {
		ch.lg.Error(op+"Invalid interaction type: ", fmt.Errorf("%v", i.Type))
		ch.Error(s, i, fmt.Errorf("Invalid interaction type: %s", i.Type.String()))
		return
	}

	ch.ClearQueue()

	ch.Success(s, i, "Cleared queue")
	ch.lg.Info("Successfully cleared queue")
}

func (ch *CommandHandler) handlePauseResume(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handlePauseResume: "
	if i.Type != discordgo.InteractionApplicationCommand {
		ch.lg.Error(op+"Invalid interaction type: ", fmt.Errorf("%v", i.Type))
		ch.Error(s, i, fmt.Errorf("Invalid interaction type: %s", i.Type.String()))
		return
	}

	if ch.voiceConn == nil {
		ch.lg.Error(op + "Not in voice channel")
		ch.Error(s, i, fmt.Errorf("Not in voice channel"))
		return
	}

	if ch.IsEmpty() {
		ch.lg.Error(op + "Queue is empty")
		ch.Error(s, i, fmt.Errorf("Queue is empty"))
		return
	}

	if ch.isSpeaking {
		ch.PausePlayback()
		ch.lg.Info("Paused playback")
		ch.Success(s, i, "Paused playback")
	} else {
		ch.ResumePlayback()
		ch.lg.Info("Resumed playback")
		ch.Success(s, i, "Resumed playback")
	}
}

func (ch *CommandHandler) handleSkip(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const op string = "handleSkip: "

	if i.Type != discordgo.InteractionApplicationCommand {
		ch.lg.Error(op+"Invalid interaction type: ", fmt.Errorf("%v", i.Type))
		ch.Error(s, i, fmt.Errorf("Invalid interaction type: %s", i.Type.String()))
		return
	}

	if ch.voiceConn == nil {
		ch.lg.Error(op + "Not in voice channel")
		ch.Error(s, i, fmt.Errorf("Not in voice channel"))
		return
	}

	if ch.IsEmpty() {
		ch.lg.Error(op + "Queue is empty")
		ch.Error(s, i, fmt.Errorf("Queue is empty"))
		return
	}

	ch.SkipSong()

	ch.Success(s, i, "Skipped")
	ch.lg.Info("Successfully skipped song")
}
