package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/models"
)

type pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type checksResponse struct {
	Data []models.CheckResult `json:"data"`
	pagination
}

type getSitesResponse struct {
	Data []models.SiteLastCheck `json:"data"`
	pagination
}

func (s *Server) handleGetSites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	page := getPage(r)
	limit := getLimit(r)

	sites, err := s.db.GetSitesWithLastCheck(page, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load sites: %v", err), http.StatusInternalServerError)
		return
	}

	count, err := s.db.CountSites()
	if err != nil {
		log.Printf("count sites %v", err)
		http.Error(w, "failed to count sites", http.StatusInternalServerError)
		return
	}

	encoder.Encode(getSitesResponse{
		Data: sites,
		pagination: pagination{
			Page:  page,
			Limit: limit,
			Total: count,
		},
	})
}

func (s *Server) handlePostSites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var input struct {
		URL       string `json:"url"`
		BodyMatch string `json:"body_match"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.AddSite(input.URL, input.BodyMatch); err != nil {
		if errors.Is(err, database.ErrInvalidURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to create site", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handlePutSites(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL       string `json:"url"`
		BodyMatch string `json:"body_match"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id, err := getIdFromRoute(r)
	if err != nil {
		http.Error(w, "invalid request id", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateSite(id, input.URL, input.BodyMatch); err != nil {
		if errors.Is(err, database.ErrInvalidURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("update site %d: %v", id, err)
		http.Error(w, "Failed to update site", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteSites(w http.ResponseWriter, r *http.Request) {
	stringId := r.PathValue("id")
	id, err := strconv.Atoi(stringId)
	if err != nil {
		http.Error(w, "invalid request id", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteSite(id); err != nil {
		log.Printf("delete site %d: %v", id, err)
		http.Error(w, "failed to delete site", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetChecks(w http.ResponseWriter, r *http.Request) {
	id, err := getIdFromRoute(r)
	if err != nil {
		http.Error(w, "invalid request id", http.StatusBadRequest)
		return
	}

	page := getPage(r)
	limit := getLimit(r)

	checks, err := s.db.GetChecks(id, page, limit)
	if err != nil {
		log.Printf("get site checks %d: %v", id, err)
		http.Error(w, "failed to get checks", http.StatusInternalServerError)
		return
	}

	count, err := s.db.CountChecks(id)
	if err != nil {
		log.Printf("count site checks %d: %v", id, err)
		http.Error(w, "failed to count checks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checksResponse{
		Data: checks,
		pagination: pagination{
			Page:  page,
			Limit: limit,
			Total: count,
		},
	})
}

func getIdFromRoute(r *http.Request) (int, error) {
	stringId := r.PathValue("id")

	return strconv.Atoi(stringId)
}

func getPage(r *http.Request) int {
	page := 1
	stringPage := r.URL.Query().Get("page")
	if stringPage != "" {
		intPage, err := strconv.Atoi(stringPage)
		if err != nil {
			return page
		}
		page = intPage
	}

	if page < 1 {
		page = 1
	}

	return page
}

func getLimit(r *http.Request) int {
	limit := 15
	stringLimit := r.URL.Query().Get("limit")
	if stringLimit != "" {
		intLimit, err := strconv.Atoi(stringLimit)
		if err != nil {
			return limit
		}
		limit = intLimit
	}

	if limit < 1 {
		limit = 15
	}

	if limit > 100 {
		limit = 100
	}

	return limit
}
