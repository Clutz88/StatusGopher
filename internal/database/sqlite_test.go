package database

import (
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/Clutz88/StatusGopher/internal/models"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com", false},
		{"no scheme", "example.com", true},
		{"ftp scheme", "ftp://example.com", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, ErrInvalidURL) {
				t.Errorf("validateURL(%q) error = %v, want ErrInvalidURL", tt.input, err)
			}
		})
	}
}

func TestAddSite(t *testing.T) {
	db := newTestDB(t)
	url := "https://example.com"
	if err := db.AddSite(url, ""); err != nil {
		t.Fatalf("AddSite(%q) error = %v", url, err)
	}

	sites, err := db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}

	if sites[0].URL != url {
		t.Errorf("GetSites() expected url %s got %s", url, sites[0].URL)
	}
}

func TestAddSite_InvalidURL(t *testing.T) {
	db := newTestDB(t)
	url := "ftp://example.com"
	err := db.AddSite(url, "")

	if !errors.Is(err, ErrInvalidURL) {
		t.Errorf("AddSite(%q) error = %v, expected ErrInvalidURL", url, err)
	}
}

func TestAddSite_Duplicate(t *testing.T) {
	db := newTestDB(t)
	url := "https://example.com"
	if err := db.AddSite(url, ""); err != nil {
		t.Fatalf("AddSite(%q) error = %v", url, err)
	}

	if err := db.AddSite(url, ""); err != nil {
		t.Fatalf("AddSite(%q) error = %v", url, err)
	}

	sites, err := db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 1 {
		t.Errorf("expected 1 site, got %d", len(sites))
	}
}

func TestDeleteSite(t *testing.T) {
	db := newTestDB(t)
	url := "https://example.com"
	if err := db.AddSite(url, ""); err != nil {
		t.Fatalf("AddSite(%q) error = %v", url, err)
	}

	sites, err := db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}

	if err := db.DeleteSite(sites[0].ID); err != nil {
		t.Fatalf("DeleteSite(%q) error = %v", sites[0].ID, err)
	}

	sites, err = db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 0 {
		t.Fatalf("expected 0 site, got %d", len(sites))
	}

}

func TestSaveResults(t *testing.T) {
	db := newTestDB(t)
	url := "https://example.com"
	if err := db.AddSite(url, ""); err != nil {
		t.Fatalf("AddSite(%q) error = %v", url, err)
	}
	sites, err := db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}
	site := sites[0]
	results := []models.CheckResult{{SiteID: site.ID, StatusCode: http.StatusOK}}

	if err := db.SaveResults(results); err != nil {
		t.Fatalf("SaveResults(%v) error = %v", results, err)
	}

	checks, err := db.GetChecks(site.ID, 1, 15)
	if err != nil {
		t.Fatalf("GetChecks() error = %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 site, got %d", len(checks))
	}

	if checks[0].StatusCode != http.StatusOK {
		t.Errorf("GetSites() expected status code %d got %d", http.StatusOK, checks[0].StatusCode)
	}
}

func newTestDB(t *testing.T) *DB {
	t.Helper() // marks this as a helper so failures point to the caller, not here
	db, err := NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("newTestDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
