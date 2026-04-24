package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	DBPath     string
	NumWorkers int
	Interval   time.Duration
	APIAddr    string
}

// Load returns a Config populated from environment variables, falling back to defaults.
func Load() *Config {
	dbPath := "./data/gopher.db"
	if v, ok := os.LookupEnv("STATUS_GOPHER_DB_PATH"); ok {
		dbPath = v
	}

	workers := 3
	if v, ok := os.LookupEnv("STATUS_GOPHER_WORKERS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			workers = n
		}
	}

	interval := 60
	if v, ok := os.LookupEnv("STATUS_GOPHER_INTERVAL"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			interval = n
		}
	}

	apiAddr := ":8080"
	if v, ok := os.LookupEnv("STATUS_GOPHER_API_ADDR"); ok {
		apiAddr = v
	}

	return &Config{
		DBPath:     dbPath,
		NumWorkers: workers,
		Interval:   time.Duration(interval) * time.Second,
		APIAddr:    apiAddr,
	}
}
