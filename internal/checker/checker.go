package checker

import (
	"net/http"
	"time"

	"github.com/Clutz88/StatusGopher/internal/models"
)

func Check(site models.Site) models.CheckResult {
	result := models.CheckResult{
		SiteID:    site.ID,
		CheckedAt: time.Now(),
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	start := time.Now()

	resp, err := client.Head(site.URL)
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
