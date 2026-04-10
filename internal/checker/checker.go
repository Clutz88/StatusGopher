package checker

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Clutz88/StatusGopher/internal/models"
)

const (
	httpTimeout = 10 * time.Second
	idleConns   = 100
	idleTimeout = 90 * time.Second
)

var DefaultClient = &http.Client{
	Timeout: httpTimeout,
	Transport: &http.Transport{
		MaxIdleConns:    idleConns,
		IdleConnTimeout: idleTimeout,
	},
}

func Check(ctx context.Context, site models.Site, client *http.Client) models.CheckResult {
	result := models.CheckResult{
		SiteID:    site.ID,
		CheckedAt: time.Now(),
	}

	start := time.Now()

	method := http.MethodHead
	if site.BodyMatch != "" {
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, site.URL, nil)
	if err != nil {
		result.Err = err.Error()
		return result
	}

	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Err = err.Error()
		result.StatusCode = 0
		return result
	}

	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode

	if site.BodyMatch != "" {
		limited := io.LimitReader(resp.Body, 1<<20) // 1MB cap
		body, err := io.ReadAll(limited)
		if err != nil {
			log.Printf("Failed to read body: %v", err)
			return result
		}
		if bodyMatch := strings.Contains(string(body), site.BodyMatch); bodyMatch == false {
			result.Err = fmt.Sprintf("body does not contain: %q", site.BodyMatch)
		}
	}

	return result
}
