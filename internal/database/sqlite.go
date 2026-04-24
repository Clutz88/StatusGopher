package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Clutz88/StatusGopher/internal/models"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // Import the driver (blank import)
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB wraps the SQLite connection used by the app.
type DB struct {
	conn *sql.DB
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// NewDB opens the database at path, runs pending migrations, and returns the DB handle.
func NewDB(path string) (*DB, error) {
	err := os.MkdirAll(filepath.Dir(path), 0750)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	err = goose.SetDialect("sqlite3")
	if err != nil {
		return nil, err
	}
	goose.SetBaseFS(migrations)
	err = goose.Up(db, "migrations")
	if err != nil {
		return nil, err
	}

	return &DB{conn: db}, nil
}

// SaveResults persists a CheckResult into the database
func (db *DB) SaveResults(results []models.CheckResult) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	placeholders := strings.Repeat("(?, ?, ?, ?, ?),", len(results))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]any, 0, len(results)*5)
	for _, res := range results {
		args = append(args, res.SiteID, res.StatusCode, res.Latency.Milliseconds(), res.CheckedAt, res.Err)
	}

	//nolint:gosec
	_, err = tx.Exec(`INSERT INTO checks (site_id, status_code, latency_ms, checked_at, error_msg) VALUES `+placeholders, args...)
	if err != nil {
		return err
	}

	err = db.updateLastCheckedAt(results, tx)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) updateLastCheckedAt(results []models.CheckResult, tx *sql.Tx) error {
	updatePlaceholders := strings.Repeat("?,", len(results))
	updatePlaceholders = updatePlaceholders[:len(updatePlaceholders)-1] // trim trailing comma

	args := make([]any, len(results))
	for i, site := range results {
		args[i] = site.SiteID
	}

	//nolint:gosec
	_, err := tx.Exec(`
			UPDATE sites SET last_checked_at = (
				SELECT MAX(checked_at) FROM checks WHERE site_id = sites.id
			)
			WHERE id IN (`+updatePlaceholders+`)
		`,
		args...,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetSites returns all sites from the database
func (db *DB) GetSites() ([]models.Site, error) {
	rows, err := db.conn.Query("SELECT id, url, body_match, added_at, last_checked_at FROM sites")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var s models.Site
		var checkedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.URL, &s.BodyMatch, &s.AddedAt, &checkedAt); err != nil {
			return nil, err
		}
		s.LastCheckedAt = checkedAt.Time
		sites = append(sites, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

// GetSitesBatch returns a section of sites using the cursor and limit
func (db *DB) GetSitesBatch(cursor, limit int) ([]models.Site, error) {
	rows, err := db.conn.Query(
		"SELECT id, url, body_match, added_at, last_checked_at FROM sites WHERE id > ? ORDER BY id ASC LIMIT ?",
		cursor,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sites := make([]models.Site, 0, limit)
	for rows.Next() {
		var s models.Site
		var checkedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.URL, &s.BodyMatch, &s.AddedAt, &checkedAt); err != nil {
			return nil, err
		}
		s.LastCheckedAt = checkedAt.Time
		sites = append(sites, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

// GetSitesWithLastCheck returns a page of sites with their last check
func (db *DB) GetSitesWithLastCheck(page, limit int) ([]models.SiteLastCheck, error) {
	rows, err := db.conn.Query(`
		SELECT s.id, s.url, s.added_at, s.body_match, c.check_id, c.status_code, c.latency_ms, c.checked_at, c.error_msg
		FROM sites s
		LEFT JOIN (
			SELECT id as check_id, site_id, status_code, latency_ms, checked_at, error_msg,
				ROW_NUMBER() OVER (PARTITION BY site_id ORDER BY checked_at DESC) as rn
			FROM checks
		) c ON c.site_id = s.id AND c.rn = 1 
		LIMIT ?
		OFFSET ?`,
		limit,
		(page-1)*limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sites := make([]models.SiteLastCheck, 0, limit)
	for rows.Next() {
		var s models.SiteLastCheck
		var checkID sql.NullInt64
		var statusCode sql.NullInt64
		var latencyMs sql.NullInt64
		var checkedAt sql.NullTime
		var errMsg sql.NullString

		if err := rows.Scan(&s.ID, &s.URL, &s.AddedAt, &s.BodyMatch, &checkID, &statusCode, &latencyMs, &checkedAt, &errMsg); err != nil {
			return nil, err
		}

		if statusCode.Valid {
			s.LastCheck = &models.CheckResult{
				ID:         int(checkID.Int64),
				SiteID:     s.ID,
				StatusCode: int(statusCode.Int64),
				Latency:    time.Duration(latencyMs.Int64) * time.Millisecond,
				CheckedAt:  checkedAt.Time,
				Err:        errMsg.String,
			}
			s.IsDown = s.LastCheck.StatusCode < 200 || s.LastCheck.StatusCode > 299 || s.LastCheck.Err != ""
		}
		if s.LastCheck != nil {
			s.LastCheckedAt = s.LastCheck.CheckedAt
		}
		sites = append(sites, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

// CountSites returns a count of all sites
func (db *DB) CountSites() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM sites").Scan(&count)
	return count, err
}

// GetSite returns a single site using the id
func (db *DB) GetSite(id int) (models.Site, error) {
	var s models.Site
	var checkedAt sql.NullTime
	err := db.conn.QueryRow("SELECT id, url, body_match, added_at, last_checked_at FROM sites WHERE id = ?", id).Scan(&s.ID, &s.URL, &s.BodyMatch, &s.AddedAt, &checkedAt)

	s.LastCheckedAt = checkedAt.Time

	return s, err
}

// AddSite persists a site to the database
func (db *DB) AddSite(url, bodyMatch string) error {
	if err := validateURL(url); err != nil {
		return err
	}

	_, err := db.conn.Exec("INSERT OR IGNORE INTO sites (url, body_match, added_at) VALUES (?, ?, ?)", url, bodyMatch, time.Now())
	return err
}

// DeleteSite removes a site from the database
func (db *DB) DeleteSite(id int) error {
	_, err := db.conn.Exec("DELETE FROM sites WHERE id = ?", id)
	return err
}

// UpdateSite persists changes to a site to the database
func (db *DB) UpdateSite(id int, newURL, bodyMatch string) error {
	if err := validateURL(newURL); err != nil {
		return err
	}

	_, err := db.conn.Exec("UPDATE sites SET url = ?, body_match = ? WHERE id = ?", newURL, bodyMatch, id)
	return err
}

// GetChecks returns a page of checks for a site
func (db *DB) GetChecks(id, page, limit int) ([]models.CheckResult, error) {
	rows, err := db.conn.Query(
		"SELECT id, site_id, status_code, latency_ms, checked_at, error_msg FROM checks WHERE site_id = ? ORDER BY checked_at DESC LIMIT ? OFFSET ?",
		id,
		limit,
		(page-1)*limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	checks := make([]models.CheckResult, 0, limit)
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

// CountChecks returns the number of checks for a site
func (db *DB) CountChecks(id int) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM checks WHERE site_id = ?", id).Scan(&count)
	return count, err
}

// SeedDB adds some example sites to the database
func (db *DB) SeedDB() {
	initialSites := []string{"https://google.com", "https://github.com", "https://go.dev", "https://google.co.uk", "https://example.com", "https://boot.dev"}
	for _, url := range initialSites {
		if err := db.AddSite(url, ""); err != nil {
			slog.Warn("could not add site", "url", url, "err", err)
		}
	}

	for i := 1; i <= 1000; i++ {
		url := fmt.Sprintf("https://test-site-%d.example.com", i)
		if err := db.AddSite(url, ""); err != nil {
			slog.Warn("could not add site", "url", url, "err", err)
		}
	}
}

// ErrInvalidURL indicates a site URL failed validation.
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
