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

func (db *DB) GetSitesWithLastCheck() ([]models.SiteLastCheck, error) {
	rows, err := db.conn.Query(`
		SELECT s.id, s.url, s.added_at, c.check_id, c.status_code, c.latency_ms, c.checked_at, c.error_msg
		FROM sites s
		LEFT JOIN (
			SELECT id as check_id, site_id, status_code, latency_ms, checked_at, error_msg,
				ROW_NUMBER() OVER (PARTITION BY site_id ORDER BY checked_at DESC) as rn
			FROM checks
		) c ON c.site_id = s.id AND c.rn = 1
	`)
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

		if err := rows.Scan(&s.ID, &s.URL, &s.AddedAt, &checkId, &statusCode, &latencyMs, &checkedAt, &errMsg); err != nil {
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
		}
		sites = append(sites, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func (db *DB) GetSite(id int) (models.Site, error) {
	var site models.Site
	err := db.conn.QueryRow("SELECT id, url, added_at FROM sites WHERE id = ?", id).Scan(&site.ID, &site.URL, &site.AddedAt)

	return site, err
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
