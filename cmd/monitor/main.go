package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Clutz88/StatusGopher/internal/config"
	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/monitor"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("connect to DB: %w", err)
	}
	defer db.Close()
	db.SeedDB()

	m := monitor.NewMonitor(db, cfg)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	fmt.Println("StatusGopher service started. Press Ctrl+C to stop.")

	m.ExecuteBatch(ctx)

	for {
		select {
		case <-sigChan:
			fmt.Println("Shutting down gracefully...")
			return nil
		case <-ticker.C:
			m.ExecuteBatch(ctx)
		}
	}
}
