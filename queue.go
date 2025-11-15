package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type CommandHandler struct {
	mu         sync.RWMutex
	queue      []*Song
	lg         *logger
	voiceConn  *discordgo.VoiceConnection
	inVC       bool
	isSpeaking bool
	skipChan   chan struct{}
	pauseChan  chan struct{}
	ctx        context.Context
}

func NewCommandHandler(logger *logger) *CommandHandler {
	return &CommandHandler{
		sync.RWMutex{},
		make([]*Song, 0),
		logger,
		nil,
		false,
		false,
		make(chan struct{}),
		make(chan struct{}),
		context.Background(),
	}
}

func (ch *CommandHandler) AddSong(url url.URL, id string) (string, error) {
	title, err := GetSongTitle(id)
	if err != nil {
		return "", fmt.Errorf("failed to get song title: %w", err)
	}

	audioPath := "audio/" + id + ".dca"

	_, err = os.Stat(audioPath)
	if err != nil {
		audioPath, err = DownloadSong(url, id)
		if err != nil {
			return "", fmt.Errorf("failed download the song: %w", err)
		}
	}

	song := NewSong(title, id, audioPath)
	ch.AppendSong(song)

	return title, nil
}

func (ch *CommandHandler) RemoveSong(index int) (string, error) {
	if index <= 0 || index > len(ch.queue) {
		return "", fmt.Errorf("index out of range: %d", index)
	}

	title := ch.queue[index-1].title

	ch.mu.Lock()
	ch.queue = append(ch.queue[:index-1], ch.queue[index:]...)
	ch.mu.Unlock()

	return title, nil
}

func (ch *CommandHandler) AppendSong(song *Song) {
	ch.mu.Lock()
	ch.queue = append(ch.queue, song)
	ch.mu.Unlock()
}

func (ch *CommandHandler) ClearQueue() {
	ch.mu.Lock()
	ch.queue = make([]*Song, 0)
	ch.mu.Unlock()
}

func (ch *CommandHandler) GetCurrentSong() *Song {
	return ch.queue[0]
}

func (ch *CommandHandler) GetSongQueue() []*Song {
	return ch.queue
}

func (ch *CommandHandler) GetFormattedQueue() string {
	songs := ch.GetSongQueue()

	if len(songs) == 0 {
		return "No songs in SongQueue"
	}

	b := strings.Builder{}
	b.Grow(len(songs) * 100)

	b.WriteString("Currently playing:\n")
	for i, song := range songs {
		if i == 0 {
			b.WriteString(fmt.Sprintf("%d. %s <--\n", i+1, song.title))
		} else {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, song.title))
		}
	}

	return b.String()
}

func (ch *CommandHandler) Shuffle() {
	ch.mu.Lock()
	for i := len(ch.queue) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		ch.queue[i], ch.queue[j] = ch.queue[j], ch.queue[i]
	}
	ch.mu.Unlock()
}

func (ch *CommandHandler) IsEmpty() bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	return len(ch.queue) == 0
}

func (ch *CommandHandler) PlaySong() {
	if ch.isSpeaking {
		ch.lg.Error("Already playing")
		return
	}

	if ch.IsEmpty() {
		ch.lg.Error("no songs in queue")
		return
	}

	song := ch.GetCurrentSong()

	err := song.LoadSound()
	if err != nil {
		ch.lg.Error("Error loading audio file: %w", err)
		return
	}

	err = ch.voiceConn.Speaking(true)
	if err != nil {
		ch.lg.Error("Error starting speaking: %w", err)
		return
	}

	ch.isSpeaking = true

	ch.lg.Info("Playing song: %s", song.title)

loop:
	for _, buff := range song.buffer {
		select {
		case <-ch.skipChan:
			break loop
		case <-ch.pauseChan:
			ch.isSpeaking = false
		inner:
			select {
			case <-ch.pauseChan:
				ch.isSpeaking = true
				break inner
			case <-ch.skipChan:
				break loop
			}
		default:
			if ch.voiceConn != nil && ch.isSpeaking {
				ch.voiceConn.OpusSend <- buff
			}
		}
	}

	err = ch.voiceConn.Speaking(false)
	if err != nil {
		ch.lg.Error("Error setting voice to speaking: %w", err)
		return
	}

	ch.isSpeaking = false

	_, err = ch.RemoveSong(1)
	if err != nil {
		ch.lg.Error("Error removing song: %w", err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	if ch.IsEmpty() {
		return
	}

	go ch.PlaySong()
}

func (ch *CommandHandler) PausePlayback() {
	ch.pauseChan <- struct{}{}
}

func (ch *CommandHandler) ResumePlayback() {
	ch.pauseChan <- struct{}{}
}

func (ch *CommandHandler) SkipSong() {
	ch.skipChan <- struct{}{}
}

func (ch *CommandHandler) HandleFileAttachment(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if len(i.ApplicationCommandData().Options) == 0 {
		return errors.New("no options provided")
	}

	attachmentID, _ := i.ApplicationCommandData().Options[0].Value.(string)
	attachmentURL := i.ApplicationCommandData().Resolved.Attachments[attachmentID].URL
	attachmentName := i.ApplicationCommandData().Resolved.Attachments[attachmentID].Filename

	ch.lg.Info("Downloading attachment: %s", attachmentName)

	song, err := downloadAttachment(attachmentURL)
	if err != nil {
		return fmt.Errorf("Error downloading attachment: %w", err)
	}

	ch.AppendSong(song)

	ch.Success(s, i, "Added to queue")
	ch.lg.Info("Added song to queue: %s", song.title)

	return nil
}

func (ch *CommandHandler) HandleYouTubeURL(_ *discordgo.Session, i *discordgo.InteractionCreate) error {
	songURL := i.ApplicationCommandData().Options[0].StringValue()
	u, err := url.Parse(songURL)
	if err != nil {
		return fmt.Errorf("Error parsing URL: %w", err)
	}

	if !IsYouTubeURL(u) {
		return fmt.Errorf("invalid YT link: %s", songURL)
	}

	ids, err := GetSongID(*u)
	if err != nil {
		return fmt.Errorf("Error getting song ID: %w", err)
	}

	if len(ids) == 1 {
		var title string

		title, err = ch.AddSong(*u, ids[0])
		if err != nil {
			return fmt.Errorf("Error adding song: %w", err)
		}

		ch.lg.Info("Successfully added: %s", title)

		return nil
	}

	var wg sync.WaitGroup

	for _, id := range ids {
		wg.Add(1)

		time.Sleep(200 * time.Millisecond)

		go func() {
			if _, err = ch.AddSong(*u, id); err != nil {
				ch.lg.Error("Error adding song: ", err)
			}

			ch.lg.Info("Added song: %s", id)

			wg.Done()
		}()
	}
	wg.Wait()

	ch.lg.Info("Successfully added: %d songs", len(ids))

	return nil
}
