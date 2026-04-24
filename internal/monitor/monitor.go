package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Clutz88/StatusGopher/internal/checker"
	"github.com/Clutz88/StatusGopher/internal/config"
	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/models"
	"github.com/google/uuid"
)

// Monitor runs checks against the list of sites
type Monitor struct {
	db         *database.DB
	mu         sync.Mutex
	numWorkers int
}

// NewMonitor constructs the Monitor
func NewMonitor(db *database.DB, cfg *config.Config) *Monitor {
	return &Monitor{db: db, numWorkers: cfg.NumWorkers}
}

// ExecuteBatch runs checks against a batch of sites
func (m *Monitor) ExecuteBatch(ctx context.Context) {
	batchLogger := slog.Default().With("batch_id", uuid.NewString())
	if m.mu.TryLock() {
		go func() {
			defer m.mu.Unlock()
			batchLogger.Info("starting check batch")
			if err := runMonitorCycle(ctx, m.db, m.numWorkers, batchLogger); err != nil {
				batchLogger.Warn("Monitor cycle failed", "err", err)
			}
		}()
	} else {
		batchLogger.Warn("Previous batch still running. Skipping this interval...")
	}

}

func runMonitorCycle(ctx context.Context, db *database.DB, numWorkers int, logger *slog.Logger) error {
	const batchSize = 100
	cursor := 0
	batchStart := time.Now()

	jobs := make(chan models.Site, numWorkers*2)
	results := make(chan models.CheckResult, numWorkers*2)

	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(ctx, w, jobs, results, &wg, logger)
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
				logger.Info("Saving to db")
				if err := db.SaveResults(batch); err != nil {
					logger.Warn("Batch save failed", "err", err)
				}
				batch = batch[:0]
			}
		}

		if len(batch) > 0 {
			logger.Info("Saving to db")
			if err := db.SaveResults(batch); err != nil {
				logger.Warn("Batch save failed", "err", err)
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
	logger.Info("Batch complete", "duration", time.Since(batchStart))
	return nil
}

func worker(ctx context.Context, id int, jobs <-chan models.Site, results chan<- models.CheckResult, wg *sync.WaitGroup, logger *slog.Logger) {
	defer wg.Done()
	for site := range jobs {
		logger.Info("worker checking site", "worker_id", id, "url", site.URL)
		results <- checker.Check(ctx, site, checker.DefaultClient)
	}
}
