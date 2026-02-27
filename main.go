package main

import (
	"fmt"
	"sync"

	"github.com/Clutz88/StatusGopher/internal/checker"
	"github.com/Clutz88/StatusGopher/internal/models"
)

func main() {
	sites := []models.Site{
		{ID: 1, URL: "https://google.com"},
		{ID: 2, URL: "https://github.com"},
		{ID: 3, URL: "https://httpbin.org/delay/2"},
		{ID: 4, URL: "https://invalid-url-example.test"},
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
	for res := range results {
		fmt.Printf("Site %d | Status: %d | Latency: %v | Err: %s\n", res.SiteID, res.StatusCode, res.Latency, res.Err)
	}
}

func worker(id int, jobs <-chan models.Site, results chan<- models.CheckResult, wg *sync.WaitGroup) {
	defer wg.Done()
	for site := range jobs {
		fmt.Printf("Worker %d checking %s...\n", id, site.URL)
		results <- checker.Check(site)
	}
}
