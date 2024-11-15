package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
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

	isPaused   bool
	isPlaying  bool
	isSpeaking bool
	Skip       bool
)

type SongQueue struct {
	mu    sync.RWMutex
	songs []*Song
}

func NewSongQueue() *SongQueue {
	return &SongQueue{songs: make([]*Song, 0, 0)}
}

func (q *SongQueue) AddSong(url url.URL, id string) (string, error) {
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
	Queue.mu.Lock()
	q.songs = append(q.songs, song)
	Queue.mu.Unlock()

	return title, nil
}

func (q *SongQueue) RemoveSong(index int) (string, error) {
	if index <= 0 || index > len(q.songs) {
		return "", fmt.Errorf("index out of range: %d", index)
	}

	title := q.songs[index-1].title

	q.mu.Lock()
	q.songs = append(q.songs[:index-1], q.songs[index:]...)
	q.mu.Unlock()

	return title, nil
}

func (q *SongQueue) Empty() {
	q.mu.Lock()
	q.songs = make([]*Song, 0, 0)
	q.mu.Unlock()
}

func (q *SongQueue) GetCurrentSong() *Song {
	return q.songs[0]
}

func (q *SongQueue) GetSongQueue() []*Song {
	return q.songs
}

func (q *SongQueue) FormatQueue() string {
	songs := q.GetSongQueue()

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

func (q *SongQueue) Shuffle() {
	q.mu.Lock()
	for i := len(q.songs) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		q.songs[i], q.songs[j] = q.songs[j], q.songs[i]
	}
	q.mu.Unlock()
}

func (q *SongQueue) IsEmpty() bool {
	return len(q.songs) == 0
}

func (q *SongQueue) PlaySong() {
	if isSpeaking {
		return
	}

	song := q.GetCurrentSong()

	if err := q.LoadSound(); err != nil {
		log.Println("Error loading audio file:", err)
		return
	}

	if err := VC.Speaking(true); err != nil {
		log.Println("Error setting voice to speaking:", err)
		return
	}

	isSpeaking = true

	log.Println("Playing song:", song.title)

loop:
	for _, buff := range song.buffer {
		select {
		case <-pauseChan:
			if q.IsEmpty() {
				break loop
			}
			if Skip {
				Skip = false
				break loop
			}
			isPlaying = false
			isPaused = true
			// Wait for resume signal
			<-resumeChan
			log.Println("Pausing")
			if q.IsEmpty() {
				break loop
			}
			if Skip {
				Skip = false
				break loop
			}
			isPlaying = true
			isPaused = false
			log.Println("Resuming")
		default:
			if !isPaused {
				isPlaying = true
				VC.OpusSend <- buff
			}
		}
	}

	if err := VC.Speaking(false); err != nil {
		log.Println("Error setting voice to speaking:", err)
	} else {
		isSpeaking = false
	}

	// Cleanup
	isPlaying = false
	q.mu.Lock()
	q.RemoveSong(1)
	q.mu.Unlock()

	time.Sleep(500 * time.Millisecond)

	if q.IsEmpty() {
		return
	}

	q.PlaySong()
}

func (q *SongQueue) PausePlayback() {
	pauseChan <- true
}

func (q *SongQueue) ResumePlayback() {
	resumeChan <- true
}

func (q *SongQueue) SkipSong() {
	q.PausePlayback()
	isPlaying = false
	isPaused = false
	q.RemoveSong(1)
	if q.IsEmpty() {
		return
	}
	isSpeaking = false
	q.PlaySong()
}

func (q *SongQueue) LoadSound() error {
	if len(q.songs) == 0 {
		return fmt.Errorf("no songs in queue")
	}
	song := q.GetCurrentSong()
	file, err := os.Open(song.audioPath)
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

		song.buffer = append(song.buffer, InBuf)
	}
}
