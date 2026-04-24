package models

import (
	"encoding/json"
	"time"
)

// Site stores data for a website that requires checks
type Site struct {
	ID            int       `json:"id"`
	URL           string    `json:"url"`
	BodyMatch     string    `json:"body_match"`
	AddedAt       time.Time `json:"added_at"`
	LastCheckedAt time.Time `json:"last_checked_at"`
}

// CheckResult stores data for a single check of a Site
type CheckResult struct {
	ID         int           `json:"id"`
	SiteID     int           `json:"site_id"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"-"`
	CheckedAt  time.Time     `json:"checked_at"`
	Err        string        `json:"error"`
}

// SiteLastCheck adds LastCheck and IsDown fields to Site
type SiteLastCheck struct {
	Site
	LastCheck *CheckResult `json:"last_check"`
	IsDown    bool         `json:"is_down"`
}

// Monitor is the contract a site checker must satisfy.
type Monitor interface {
	Check(site Site) CheckResult
}

// MarshalJSON implements json.Marshaler, serialising Latency as latency_ms (milliseconds).
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
