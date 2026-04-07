package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Clutz88/StatusGopher/internal/checker"
	"github.com/Clutz88/StatusGopher/internal/config"
	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/models"
)

type Monitor struct {
	db         *database.DB
	mu         sync.Mutex
	numWorkers int
}

func NewMonitor(db *database.DB, cfg *config.Config) *Monitor {
	return &Monitor{db: db, numWorkers: cfg.NumWorkers}
}

func (m *Monitor) executeBatch(ctx context.Context) {
	if m.mu.TryLock() {
		go func() {
			defer m.mu.Unlock()
			log.Println("Starting check batch")
			if err := runMonitorCycle(ctx, m.db, m.numWorkers); err != nil {
				log.Printf("Monitor cycle failed: %v", err)
			}
		}()
	} else {
		log.Println("Previous batch still running. Skipping this interval...")
	}

}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	db, err := database.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatal("Failed to connect to DB: ", err)
	}
	initialSites := []string{"https://google.com", "https://github.com", "https://go.dev", "https://google.co.uk", "https://example.com", "https://boot.dev"}
	for _, url := range initialSites {
		if err := db.AddSite(url); err != nil {
			log.Printf("warn: could not add site %s: %v", url, err)
		}
	}
	defer db.Close()

	m := NewMonitor(db, cfg)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.Println("StatusGopher service started. Press Ctrl+C to stop.")

	m.executeBatch(ctx)

	for {
		select {
		case <-sigChan:
			log.Println("Shutting down gracefully...")
			return
		case <-ticker.C:
			m.executeBatch(ctx)
		}
	}
}

func runMonitorCycle(ctx context.Context, db *database.DB, numWorkers int) error {
	sites, err := db.GetSites()
	if err != nil {
		return fmt.Errorf("load sites: %w", err)
	}

	jobs := make(chan models.Site, len(sites))
	results := make(chan models.CheckResult, len(sites))

	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(ctx, w, jobs, results, &wg)
	}

	for _, s := range sites {
		jobs <- s
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	log.Println("--- Monitoring Results ---")
	var allResults []models.CheckResult
	for res := range results {
		allResults = append(allResults, res)
	}

	if err := db.SaveResults(allResults); err != nil {
		log.Printf("Batch save failed: %v\n", err)
	} else {
		log.Printf("Successfully saved batch of %d results\n", len(allResults))
	}
	log.Println("Batch complete")

	return nil
}

func worker(ctx context.Context, id int, jobs <-chan models.Site, results chan<- models.CheckResult, wg *sync.WaitGroup) {
	defer wg.Done()
	for site := range jobs {
		log.Printf("Worker %d checking %s...\n", id, site.URL)
		results <- checker.Check(ctx, site)
	}
}
