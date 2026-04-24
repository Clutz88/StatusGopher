package logging

import (
	"context"
	"log/slog"
	"os"
)

type LoggerKey struct{}

func New(level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})

	return slog.New(h)
}

func FromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(LoggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
