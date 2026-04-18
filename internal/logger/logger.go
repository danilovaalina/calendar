package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Log struct {
	Method  string
	URI     string
	Latency time.Duration
	Status  int
}

type Options struct {
	FilePath   string
	BufferSize int
}

type AsyncLogger struct {
	logs   chan Log
	writer zerolog.Logger
}

func New(opts Options) *AsyncLogger {
	var output io.Writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	if opts.FilePath != "" {
		if file, err := os.OpenFile(opts.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); err == nil {
			output = file
		}
	}

	return &AsyncLogger{
		// Используем буферизированный канал, чтобы хендлеры не ждали,
		// пока логгер успеет прочитать запись
		logs:   make(chan Log, opts.BufferSize),
		writer: zerolog.New(output).With().Timestamp().Logger(),
	}
}

func (l *AsyncLogger) Log(log Log) {
	l.logs <- log
}

func (l *AsyncLogger) Run(ctx context.Context) {
	for {
		select {
		case log := <-l.logs:
			l.writer.Info().
				Str("method", log.Method).
				Str("uri", log.URI).
				Dur("latency", log.Latency).
				Int("status", log.Status).
				Msg("request processed")
		case <-ctx.Done():
			log.Info().Msg("logger is shutting down...")
			return
		}
	}
}

func (l *AsyncLogger) SetWriter(output zerolog.Logger) {
	l.writer = output
}
