package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Clutz88/StatusGopher/internal/config"
	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/monitor"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to DB: ", err)
	}
	db.SeedDB()
	defer db.Close()

	m := monitor.NewMonitor(db, cfg)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.Println("StatusGopher service started. Press Ctrl+C to stop.")

	m.ExecuteBatch(ctx)

	for {
		select {
		case <-sigChan:
			log.Println("Shutting down gracefully...")
			return
		case <-ticker.C:
			m.ExecuteBatch(ctx)
		}
	}
}
