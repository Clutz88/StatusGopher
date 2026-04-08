package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/models"
)

func TestHandleGetSites_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sites", nil)
	s := newTestServer(t)

	s.handleGetSites(w, r)

	var sites []models.Site
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

	s.handleGetSites(w, r)

	var sites []models.Site
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

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := database.NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("newTestServer: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewServer(":0", db)
}
