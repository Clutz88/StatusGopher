package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Clutz88/StatusGopher/internal/checker"
	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/models"
)

var batchMutex sync.Mutex

func main() {
	db, err := database.NewDB("./data/gopher.db")
	if err != nil {
		log.Fatal("Failed to connect to DB: ", err)
	}
	initialSites := []string{"https://google.com", "https://github.com", "https://go.dev"}
	for _, url := range initialSites {
		db.AddSite(url)
	}
	defer db.Conn.Close()

	trigger := make(chan struct{}, 1)
	trigger <- struct{}{}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.Println("StatusGopher service started. Press Ctrl+C to stop.")

	for {
		select {
		case <-sigChan:
			log.Println("Shutting down gracefully...")
			return
		case <-trigger:
			executeBatch(db)
		case <-ticker.C:
			executeBatch(db)
		}
	}
}

func executeBatch(db *database.DB) {
	if batchMutex.TryLock() {
		go func() {
			defer batchMutex.Unlock()
			log.Println("Starting check batch")
			runMonitorCycle(db)
		}()
	} else {
		log.Println("Previous batch still running. Skipping this interval...")
	}
}

func runMonitorCycle(db *database.DB) {
	sites, err := db.GetSites()
	if err != nil {
		log.Fatal("Could not load sites:", err)
	}

	jobs := make(chan models.Site, len(sites))
	results := make(chan models.CheckResult, len(sites))

	var wg sync.WaitGroup
	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	for _, s := range sites {
		jobs <- s
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	fmt.Println("--- Monitoring Results ---")
	var allResults []models.CheckResult
	for res := range results {
		allResults = append(allResults, res)
	}

	if err := db.SaveResults(allResults); err != nil {
		log.Printf("Batch save failed: %v\n", err)
	} else {
		fmt.Printf("Successfully saved batch of %d results\n", len(allResults))
	}
	log.Println("Batch complete")
}

func worker(id int, jobs <-chan models.Site, results chan<- models.CheckResult, wg *sync.WaitGroup) {
	defer wg.Done()
	for site := range jobs {
		fmt.Printf("Worker %d checking %s...\n", id, site.URL)
		results <- checker.Check(site)
	}
}
