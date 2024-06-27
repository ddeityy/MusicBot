package main

import "log"

type Song struct {
	title     string
	id        string
	audioPath string
	buffer    [][]byte
}

func NewSong(title, id, audioPath string) *Song {
	buffer := make([][]byte, 0)
	return &Song{title, id, audioPath, buffer}
}

func (s *Song) ClearBuffer() {
	s.buffer = make([][]byte, 0)
	log.Println("Cleared audio buffer")
}
