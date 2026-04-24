package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Clutz88/StatusGopher/internal/database"
)

// Server handles HTTP requests for the sites API.
type Server struct {
	db         *database.DB
	httpServer *http.Server
	routes     []route
}

type route struct {
	method  string
	path    string
	handler http.HandlerFunc
}

// NewServer returns a server with registered routes
func NewServer(addr string, db *database.DB) *Server {
	s := &Server{db: db}
	s.routes = []route{
		{"GET", "/sites", s.handleGetSites},
		{"POST", "/sites", s.handlePostSites},
		{"PUT", "/sites/{id}", s.handlePutSites},
		{"DELETE", "/sites/{id}", s.handleDeleteSites},
		{"GET", "/sites/{id}/checks", s.handleGetChecks},
	}

	mux := http.NewServeMux()

	for _, r := range s.routes {
		mux.HandleFunc(r.method+" "+r.path, r.handler)
	}

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

// Start gets the server to listen on the registered port
func (s *Server) Start() error {
	log.Printf("API server listening on %s", s.httpServer.Addr)

	log.Println("Registered routes:")
	for _, r := range s.routes {
		log.Printf("  %-7s %s", r.method, r.path)
	}

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop makes the server run its shutdown
func (s *Server) Stop(ctx context.Context) error {
	log.Println("API server shutting down...")
	return s.httpServer.Shutdown(ctx)
}
