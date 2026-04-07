package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Clutz88/StatusGopher/internal/api"
	"github.com/Clutz88/StatusGopher/internal/config"
	"github.com/Clutz88/StatusGopher/internal/database"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to DB: ", err)
	}
	defer db.Close()

	server := api.NewServer(cfg.APIAddr, db)
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan // block until signal received

	log.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
