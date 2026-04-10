package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/Clutz88/StatusGopher/internal/database"
	"github.com/Clutz88/StatusGopher/internal/models"
)

type checksResponse struct {
	Data  []models.CheckResult `json:"data"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
	Total int                  `json:"total"`
}

func (s *Server) handleGetSites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	sites, err := s.db.GetSites()
	if err != nil {
		http.Error(w, "failed to load sites", http.StatusInternalServerError)
		return
	}

	encoder.Encode(sites)
}

func (s *Server) handlePostSites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var input struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.AddSite(input.URL); err != nil {
		if errors.Is(err, database.ErrInvalidURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to create site", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
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
	stringId := r.PathValue("id")
	id, err := strconv.Atoi(stringId)
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
		Data:  checks,
		Page:  page,
		Limit: limit,
		Total: count,
	})
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
