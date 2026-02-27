package models

import "time"

type Site struct {
	ID      int       `json:"id"`
	URL     string    `json:"url"`
	AddedAt time.Time `json:"added_at"`
}

type CheckResult struct {
	ID         int           `json:"id"`
	SiteID     int           `json:"site_id"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency"`
	CheckedAt  time.Time     `json:"checked_at"`
	Err        string        `json:"error"`
}

type Monitor interface {
	Check(site Site) CheckResult
}
