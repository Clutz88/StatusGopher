package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
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

func (m *Monitor) ExecuteBatch(ctx context.Context) {
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

func runMonitorCycle(ctx context.Context, db *database.DB, numWorkers int) error {
	const batchSize = 100
	cursor := 0
	batchStart := time.Now()

	jobs := make(chan models.Site, numWorkers*2)
	results := make(chan models.CheckResult, numWorkers*2)

	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(ctx, w, jobs, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var resultWg sync.WaitGroup
	resultWg.Go(func() {
		batch := make([]models.CheckResult, 0, batchSize)
		for res := range results {
			batch = append(batch, res)
			if len(batch) >= batchSize {
				log.Println("Saving to db...")
				if err := db.SaveResults(batch); err != nil {
					log.Printf("Batch save failed: %v\n", err)
				}
				batch = batch[:0]
			}
		}

		if len(batch) > 0 {
			log.Println("Saving to db...")
			if err := db.SaveResults(batch); err != nil {
				log.Printf("Batch save failed: %v\n", err)
			}
		}
	})

	for {
		sites, err := db.GetSitesBatch(cursor, batchSize)
		if err != nil {
			return fmt.Errorf("load sites: %w", err)
		}

		for _, s := range sites {
			jobs <- s
		}

		if len(sites) < batchSize {
			break
		}
		cursor = sites[len(sites)-1].ID
	}
	close(jobs)

	resultWg.Wait()
	log.Printf("Batch complete - %s", time.Since(batchStart))
	return nil
}

func worker(ctx context.Context, id int, jobs <-chan models.Site, results chan<- models.CheckResult, wg *sync.WaitGroup) {
	defer wg.Done()
	for site := range jobs {
		log.Printf("Worker %d checking %s...\n", id, site.URL)
		results <- checker.Check(ctx, site, checker.DefaultClient)
	}
}
