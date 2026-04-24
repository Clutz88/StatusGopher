package logging

import (
	"context"
	"log/slog"
	"os"
)

// LoggerKey struct for logging
type LoggerKey struct{}

// New returns a new JSON slog logger
func New(level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})

	return slog.New(h)
}

// FromCtx builds a slog logger containing the ctx to be able to locate logs across a single request
func FromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(LoggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
