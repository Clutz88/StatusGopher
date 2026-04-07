package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/Clutz88/StatusGopher/internal/database"
)

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

	checks, err := s.db.GetChecks(id)
	if err != nil {
		log.Printf("get site checks %d: %v", id, err)
		http.Error(w, "failed to get checks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checks)
}
