package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Clutz88/StatusGopher/internal/api"
	"github.com/Clutz88/StatusGopher/internal/config"
	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/logging"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	logger := logging.New(slog.LevelInfo)
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("connect to DB: %w", err)
	}
	defer db.Close()
	db.SeedDB()

	server := api.NewServer(cfg.APIAddr, db)
	go func() {
		if err := server.Start(); err != nil {
			logging.FromCtx(ctx).Warn("server error", "err", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan // block until signal received

	slog.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	return nil
}
