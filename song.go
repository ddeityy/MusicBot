package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

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

func (s *Song) LoadSound() error {
	file, err := os.Open(s.audioPath)
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

		s.buffer = append(s.buffer, InBuf)
	}
}
