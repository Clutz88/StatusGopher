package checker

import (
	"context"
	"net/http"
	"time"

	"github.com/Clutz88/StatusGopher/internal/models"
)

const (
	httpTimeout = 10 * time.Second
	idleConns   = 100
	idleTimeout = 90 * time.Second
)

var defaultClient = &http.Client{
	Timeout: httpTimeout,
	Transport: &http.Transport{
		MaxIdleConns:    idleConns,
		IdleConnTimeout: idleTimeout,
	},
}

func Check(ctx context.Context, site models.Site) models.CheckResult {
	result := models.CheckResult{
		SiteID:    site.ID,
		CheckedAt: time.Now(),
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, site.URL, nil)
	if err != nil {
		result.Err = err.Error()
		return result
	}

	resp, err := defaultClient.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Err = err.Error()
		result.StatusCode = 0
		return result
	}

	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	return result
}
