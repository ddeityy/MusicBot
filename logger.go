package main

import (
	"fmt"
	"log/slog"
)

type logger struct {
	s *slog.Logger
}

func NewLogger() *logger {
	return &logger{s: slog.Default()}
}

func (l logger) Error(msg string, err ...error) {
	if len(err) == 0 || err == nil {
		l.s.Error(msg)
		return
	}
	l.s.Error(msg, slog.String("error", err[0].Error()))
}

func (l logger) Info(format string, arg ...any) {
	l.s.Info(fmt.Sprintf(format, arg...))
}
