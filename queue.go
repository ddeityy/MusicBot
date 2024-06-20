package musicbot

import (
	"fmt"
	URL "net/url"
)

type Queue struct {
	songs []*Song
}

func NewQueue() *Queue {
	return &Queue{songs: make([]*Song, 0, 0)}
}

func (q *Queue) AddSong(url string) error {
	u, err := URL.Parse(url)
	if err != nil {
		return fmt.Errorf("AddSong: invalid URL: %w", err)
	}

	if !IsYouTubeURL(u) {
		return fmt.Errorf("AddSong: url is not a valid youtube URL: %s", url)
	}

	title, err := GetSongTitle(*u)
	id := GetSongID(*u)
	if err != nil {
		return fmt.Errorf("AddSong: failed to get song title: %w", err)
	}

	song := &Song{URL: *u, Title: title, ID: id}

	if err := song.Download(); err != nil {
		return fmt.Errorf("AddSong: failed download the song: %w", err)
	}

	q.songs = append(q.songs, song)

	return nil
}

func (q *Queue) RemoveSong(index int) error {
	index += 1
	if index <= 0 || index > len(q.songs) {
		return fmt.Errorf("RemoveSong: index out of range: %d", index)
	}

	q.songs = append(q.songs[:index-1], q.songs[index:]...)

	return nil
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

func (q *Queue) GetLastSong() *Song {
	return q.songs[len(q.songs)-1]
}
