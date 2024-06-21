package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	URL "net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	// Channels for signaling playback control
	pauseChan  = make(chan bool)
	resumeChan = make(chan bool)

	// Channel for playback status updates
	statusChan = make(chan bool)

	skipChan = make(chan struct{})

	isPaused  bool
	isPlaying bool
)

type Queue struct {
	mu    sync.RWMutex
	songs []*Song
}

func NewQueue() *Queue {
	return &Queue{songs: make([]*Song, 0, 0)}
}

func (q *Queue) AddSong(url string) (string, error) {
	u, err := URL.Parse(url)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if !IsYouTubeURL(u) {
		return "", fmt.Errorf("url is not a valid youtube URL: %s", url)
	}

	id := GetSongID(*u)
	title, err := GetSongTitle(*u)
	if err != nil {
		return "", fmt.Errorf("failed to get song title: %w", err)
	}

	audioPath := "audio/" + id + ".dca"

	_, err = os.Stat(audioPath)
	if err != nil {
		audioPath, err = DownloadSong(id)
		if err != nil {
			return "", fmt.Errorf("failed download the song: %w", err)
		}

		if err = os.Remove(fmt.Sprintf("video/%s.mp4", id)); err != nil {
			return "", fmt.Errorf("error removing video: %s", err)
		}
	}

	song := &Song{URL: *u, Title: title, ID: id, AudioPath: audioPath}
	q.songs = append(q.songs, song)

	return song.Title, nil
}

func (q *Queue) RemoveSong(index int) (string, error) {
	if index <= 0 || index > len(q.songs) {
		return "", fmt.Errorf("index out of range: %d", index)
	}

	title := q.songs[index-1].Title

	q.songs = append(q.songs[:index-1], q.songs[index:]...)

	return title, nil
}

func (q *Queue) Empty() {
	q.songs = make([]*Song, 0, 0)
}

func (q *Queue) GetCurrentSong() *Song {
	if len(q.songs) == 0 {
		return nil
	}
	return q.songs[0]
}

func (q *Queue) GetQueue() []*Song {
	return q.songs
}

func (q *Queue) FormatQueue() string {
	songs := q.GetQueue()

	if len(songs) == 0 {
		return "No songs in queue"
	}

	b := strings.Builder{}
	b.Grow(len(songs) * 100)

	for i, song := range songs {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, song.Title))
	}

	return b.String()
}

func (q *Queue) Shuffle() {
	for i := len(q.songs) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		q.songs[i], q.songs[j] = q.songs[j], q.songs[i]
	}
}

func (q *Queue) IsEmpty() bool {
	return len(q.songs) == 0
}

func (q *Queue) PlaySong() {
	song := q.GetCurrentSong()

	if err := q.LoadSound(); err != nil {
		log.Println("Error loading audio file:", err)
		return
	}

	if err := VC.Speaking(true); err != nil {
		log.Println("Error setting voice to speaking:", err)
	}

loop:
	for _, buff := range song.Buffer {
		select {
		case <-pauseChan:
			isPaused = true
			// Wait for resume signal
			<-resumeChan
			isPaused = false
		case <-skipChan:
			break loop
		default:
			if !isPaused {
				isPlaying = true
				VC.OpusSend <- buff
			}
		}
	}

	if err := VC.Speaking(false); err != nil {
		log.Println("Error setting voice to speaking:", err)
	}

	// Cleanup
	song.ClearBuffer()
	q.mu.Lock()
	q.RemoveSong(1)
	q.mu.Unlock()

	time.Sleep(500 * time.Millisecond)

	if q.IsEmpty() {
		return
	}

	q.PlaySong()
}

func (q *Queue) PausePlayback() {
	pauseChan <- true
}

func (q *Queue) ResumePlayback() {
	resumeChan <- true
}

func (q *Queue) SkipSong() {
	skipChan <- struct{}{}
}

func (q *Queue) LoadSound() error {
	song := q.GetCurrentSong()
	file, err := os.Open(song.AudioPath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}

	var opuslen int16

	for {
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return fmt.Errorf("error closing file: %w", err)
			}
			return nil
		}

		if err != nil {
			return fmt.Errorf("Error reading file: %w", err)
		}

		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		if err != nil {
			return fmt.Errorf("Error reading file: %w", err)
		}

		song.Buffer = append(song.Buffer, InBuf)
	}
}
