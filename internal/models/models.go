package models

import (
	"encoding/json"
	"time"
)

type Site struct {
	ID      int       `json:"id"`
	URL     string    `json:"url"`
	AddedAt time.Time `json:"added_at"`
}

type CheckResult struct {
	ID         int           `json:"id"`
	SiteID     int           `json:"site_id"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"-"`
	CheckedAt  time.Time     `json:"checked_at"`
	Err        string        `json:"error"`
}

type Monitor interface {
	Check(site Site) CheckResult
}

func (c CheckResult) MarshalJSON() ([]byte, error) {
	type Alias CheckResult

	return json.Marshal(&struct {
		Alias
		Latency int64 `json:"latency_ms"`
	}{
		Alias:   Alias(c),
		Latency: c.Latency.Milliseconds(),
	})
}
