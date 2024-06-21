package main

import "net/url"

type Song struct {
	Title     string
	ID        string
	URL       url.URL
	AudioPath string
	Buffer    [][]byte
}

func (s *Song) ClearBuffer() {
	s.Buffer = make([][]byte, 0)
}
