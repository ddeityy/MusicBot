package main

type Song struct {
	Title     string
	ID        string
	AudioPath string
	Buffer    [][]byte
}

func (s *Song) ClearBuffer() {
	s.Buffer = make([][]byte, 0)
}
