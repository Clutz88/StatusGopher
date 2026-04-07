package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/Clutz88/StatusGopher/internal/models"
	_ "modernc.org/sqlite" // Import the driver (blank import)
)

type DB struct {
	conn *sql.DB
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func NewDB(path string) (*DB, error) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	err = buildSchema(db)
	if err != nil {
		return nil, err
	}

	return &DB{conn: db}, nil
}

func (db *DB) SaveResults(results []models.CheckResult) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO checks (site_id, status_code, latency_ms, checked_at, error_msg)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, res := range results {
		_, err := stmt.Exec(
			res.SiteID,
			res.StatusCode,
			res.Latency.Milliseconds(),
			res.CheckedAt,
			res.Err,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func buildSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS sites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE NOT NULL,
			added_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS checks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			status_code INTEGER,
			latency_ms INTEGER,
			checked_at DATETIME NOT NULL,
			error_msg TEXT,
			FOREIGN KEY (site_id) REFERENCES sites(id)
		);`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("Could not build schema: %w", err)
	}

	return nil
}

func (db *DB) GetSites() ([]models.Site, error) {
	rows, err := db.conn.Query("SELECT id, url, added_at FROM sites")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var s models.Site
		if err := rows.Scan(&s.ID, &s.URL, &s.AddedAt); err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func (db *DB) AddSite(url string) error {
	if err := validateURL(url); err != nil {
		return err
	}

	_, err := db.conn.Exec("INSERT OR IGNORE INTO sites (url, added_at) VALUES (?, ?)", url, time.Now())
	return err
}

func (db *DB) DeleteSite(id int) error {
	_, err := db.conn.Exec("DELETE FROM sites WHERE id = ?", id)
	return err
}

func (db *DB) UpdateSite(id int, newUrl string) error {
	if err := validateURL(newUrl); err != nil {
		return err
	}

	_, err := db.conn.Exec("UPDATE sites SET url = ? WHERE id = ?", newUrl, id)
	return err
}

func (db *DB) GetChecks(id int) ([]models.CheckResult, error) {
	rows, err := db.conn.Query("SELECT id, site_id, status_code, latency_ms, checked_at, error_msg FROM checks WHERE site_id = ? ORDER BY checked_at DESC", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	checks := []models.CheckResult{}
	for rows.Next() {
		var c models.CheckResult
		var latencyMs int64
		if err := rows.Scan(&c.ID, &c.SiteID, &c.StatusCode, &latencyMs, &c.CheckedAt, &c.Err); err != nil {
			return nil, err
		}
		c.Latency = time.Duration(latencyMs) * time.Millisecond
		checks = append(checks, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return checks, nil
}

var ErrInvalidURL = errors.New("invalid URL")

func validateURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidURL, rawURL)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: %q: scheme must be http or https", ErrInvalidURL, rawURL)
	}

	return nil
}
