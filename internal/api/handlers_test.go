package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/models"
)

func TestHandleGetSites_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sites", nil)
	s := newTestServer(t)

	s.handleGetSites(w, r)

	var sites []models.SiteLastCheck
	if err := json.NewDecoder(w.Body).Decode(&sites); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("handleGetSites(): expected %d statusCode, got %d", http.StatusOK, w.Code)
	}

	if len(sites) != 0 {
		t.Errorf("expected empty slice, got %d sites", len(sites))
	}
}

func TestHandleGetSites(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sites", nil)
	s := newTestServer(t)

	url := "http://example.com"
	if err := s.db.AddSite(url); err != nil {
		t.Fatalf("AddSite(%s) error = %v", url, err)
	}

	dbSites, err := s.db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(dbSites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(dbSites))
	}

	site := dbSites[0]

	if err := s.db.SaveResults([]models.CheckResult{{SiteID: site.ID, StatusCode: http.StatusOK, Latency: time.Duration(78 * time.Millisecond)}}); err != nil {
		t.Fatalf("SaveResults() error = %v", err)
	}

	s.handleGetSites(w, r)

	var sites []models.SiteLastCheck
	if err := json.NewDecoder(w.Body).Decode(&sites); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("handleGetSites(): expected %d statusCode, got %d", http.StatusOK, w.Code)
	}

	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d sites", len(sites))
	}

	if sites[0].URL != url {
		t.Errorf("expected first site to have url %s, got %s", url, sites[0].URL)
	}

	if sites[0].LastCheck == nil {
		t.Fatalf("expected first site to have checks got %v", sites[0].LastCheck)
	}

	if sites[0].LastCheck.StatusCode != http.StatusOK {
		t.Errorf("expected first site to have status code %d got %d", http.StatusOK, sites[0].LastCheck.StatusCode)
	}
}

func TestHandlePostSites(t *testing.T) {
	w := httptest.NewRecorder()
	body := strings.NewReader(`{"url": "https://example.com"}`)
	r := httptest.NewRequest(http.MethodPost, "/sites", body)
	s := newTestServer(t)

	s.handlePostSites(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("handleGetSites(): expected %d statusCode, got %d", http.StatusCreated, w.Code)
	}
}

func TestHandlePostSites_InvalidURL(t *testing.T) {
	w := httptest.NewRecorder()
	body := strings.NewReader(`{"url": "ftp://example.com"}`)
	r := httptest.NewRequest(http.MethodPost, "/sites", body)
	s := newTestServer(t)

	s.handlePostSites(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("handlePostSites(): expected %d statusCode, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleDeleteSite(t *testing.T) {
	w := httptest.NewRecorder()
	s := newTestServer(t)

	url := "http://example.com"
	if err := s.db.AddSite(url); err != nil {
		t.Fatalf("AddSite(%s) error = %v", url, err)
	}

	sites, err := s.db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}

	site := sites[0]
	r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/sites/%d", site.ID), nil)
	r.SetPathValue("id", fmt.Sprintf("%d", site.ID))

	s.handleDeleteSites(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("handlePostSites(): expected %d statusCode, got %d", http.StatusNoContent, w.Code)
	}
}

func TestHandleGetChecks_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	s := newTestServer(t)

	url := "http://example.com"
	if err := s.db.AddSite(url); err != nil {
		t.Fatalf("AddSite(%s) error = %v", url, err)
	}

	sites, err := s.db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}

	site := sites[0]
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sites/%d/checks", site.ID), nil)
	r.SetPathValue("id", fmt.Sprintf("%d", site.ID))

	s.handleGetChecks(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("handleGetChecks(): expected %d statusCode, got %d", http.StatusOK, w.Code)
	}

	var body checksResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Data) != 0 {
		t.Fatalf("expected empty slice, got %d", len(body.Data))
	}

	if body.Total != 0 {
		t.Errorf("expected total 0, got %d", body.Total)
	}
}
func TestHandleGetChecks_Pagination(t *testing.T) {
	w := httptest.NewRecorder()
	s := newTestServer(t)

	url := "http://example.com"
	if err := s.db.AddSite(url); err != nil {
		t.Fatalf("AddSite(%s) error = %v", url, err)
	}

	sites, err := s.db.GetSites()
	if err != nil {
		t.Fatalf("GetSites() error = %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}

	site := sites[0]
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sites/%d/checks?limit=2", site.ID), nil)
	r.SetPathValue("id", fmt.Sprintf("%d", site.ID))

	if err := s.db.SaveResults([]models.CheckResult{{SiteID: site.ID}, {SiteID: site.ID}, {SiteID: site.ID}, {SiteID: site.ID}, {SiteID: site.ID}}); err != nil {
		t.Fatalf("failed to save results %v", err)
	}

	s.handleGetChecks(w, r)
	var body checksResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(body.Data) != 2 {
		t.Fatalf("expected 2 results, got %d", len(body.Data))
	}

	if body.Total != 5 {
		t.Errorf("expected total 5, got %d", body.Total)
	}
}
func TestHandleGetChecks_InvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	s := newTestServer(t)
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sites/%s/checks", "one"), nil)
	r.SetPathValue("id", "one")

	s.handleGetChecks(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlePutSites(t *testing.T) {
	w := httptest.NewRecorder()
	s := newTestServer(t)

	if err := s.db.AddSite("https://example.com"); err != nil {
		t.Fatalf("failed to save site %v", err)
	}

	sites, err := s.db.GetSites()
	if err != nil {
		t.Fatalf("failed to get sites %v", err)
	}
	if len(sites) < 1 {
		t.Fatalf("Sites should be 1, got %d instead", len(sites))
	}
	site := sites[0]

	body := strings.NewReader(`{"url": "https://examples.com"}`)
	r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/sites/%d", site.ID), body)
	r.SetPathValue("id", fmt.Sprintf("%d", site.ID))

	s.handlePutSites(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("handlePutSites(): expected %d statusCode, got %d", http.StatusNoContent, w.Code)
	}

	updatedSite, err := s.db.GetSite(site.ID)
	if err != nil {
		t.Fatalf("failed to refetch site %v", err)
	}
	if updatedSite.URL != "https://examples.com" {
		t.Errorf("site name was not updated, expected %s, got %s", "https://examples.com", updatedSite.URL)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("newTestServer: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewServer(":0", db)
}
