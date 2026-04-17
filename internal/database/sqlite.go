package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	placeholders := strings.Repeat("?,", len(results))
	placeholders = placeholders[:len(placeholders)-1] // trim trailing comma

	args := make([]any, len(results))
	for i, site := range results {
		args[i] = site.SiteID
	}

	_, err = tx.Exec(`
			UPDATE sites SET last_checked_at = (
				SELECT MAX(checked_at) FROM checks WHERE site_id = sites.id
			)
			WHERE id IN (`+placeholders+`)
		`,
		args...,
	)
	if err != nil {
		return err
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
		);
		
		CREATE INDEX IF NOT EXISTS idx_checks_site_id ON checks(site_id);
		CREATE INDEX IF NOT EXISTS idx_checks_site_id_checked_at ON checks(site_id, checked_at DESC);
		`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("Could not build schema: %w", err)
	}

	_, err := db.Exec("ALTER TABLE sites ADD COLUMN body_match TEXT")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("migrate body_match: %w", err)
	}
	_, err = db.Exec("ALTER TABLE sites ADD COLUMN last_checked_at DATETIME")
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("migrate last_checked_at: %w", err)
	}

	return nil
}

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

	var sites []models.SiteLastCheck
	for rows.Next() {
		var s models.SiteLastCheck
		var checkId sql.NullInt64
		var statusCode sql.NullInt64
		var latencyMs sql.NullInt64
		var checkedAt sql.NullTime
		var errMsg sql.NullString

		if err := rows.Scan(&s.ID, &s.URL, &s.AddedAt, &s.BodyMatch, &checkId, &statusCode, &latencyMs, &checkedAt, &errMsg); err != nil {
			return nil, err
		}

		if statusCode.Valid {
			s.LastCheck = &models.CheckResult{
				ID:         int(checkId.Int64),
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

func (db *DB) CountSites() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM sites").Scan(&count)
	return count, err
}

func (db *DB) GetSite(id int) (models.Site, error) {
	var s models.Site
	var checkedAt sql.NullTime
	err := db.conn.QueryRow("SELECT id, url, body_match, added_at, last_checked_at FROM sites WHERE id = ?", id).Scan(&s.ID, &s.URL, &s.BodyMatch, &s.AddedAt, &checkedAt)

	s.LastCheckedAt = checkedAt.Time

	return s, err
}

func (db *DB) AddSite(url, body_match string) error {
	if err := validateURL(url); err != nil {
		return err
	}

	_, err := db.conn.Exec("INSERT OR IGNORE INTO sites (url, body_match, added_at) VALUES (?, ?, ?)", url, body_match, time.Now())
	return err
}

func (db *DB) DeleteSite(id int) error {
	_, err := db.conn.Exec("DELETE FROM sites WHERE id = ?", id)
	return err
}

func (db *DB) UpdateSite(id int, newUrl, body_match string) error {
	if err := validateURL(newUrl); err != nil {
		return err
	}

	_, err := db.conn.Exec("UPDATE sites SET url = ?, body_match = ? WHERE id = ?", newUrl, body_match, id)
	return err
}

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

func (db *DB) CountChecks(id int) (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM checks WHERE site_id = ?", id).Scan(&count)
	return count, err
}

func (db *DB) SeedDB() {
	initialSites := []string{"https://google.com", "https://github.com", "https://go.dev", "https://google.co.uk", "https://example.com", "https://boot.dev"}
	for _, url := range initialSites {
		if err := db.AddSite(url, ""); err != nil {
			log.Printf("warn: could not add site %s: %v", url, err)
		}
	}

	for i := 1; i <= 1000; i++ {
		url := fmt.Sprintf("https://test-site-%d.example.com", i)
		if err := db.AddSite(url, ""); err != nil {
			log.Printf("warn: could not add site %s: %v", url, err)
		}
	}
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
