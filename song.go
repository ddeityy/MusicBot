package main

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
