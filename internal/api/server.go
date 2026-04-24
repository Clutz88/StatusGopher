package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Clutz88/StatusGopher/internal/database"
)

// Server exposes the sites HTTP API, owning the underlying http.Server
// and the database handle its handlers read from.
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
		Handler:           traceMiddleware(logMiddleware(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

// Start begins listening on the configured address and blocks until the server is stopped.
func (s *Server) Start() error {
	slog.Info("api listening", "addr", s.httpServer.Addr)

	for _, r := range s.routes {
		slog.Info("registered route", "method", r.method, "path", r.path)
	}

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop makes the server run its shutdown
func (s *Server) Stop(ctx context.Context) error {
	slog.Info("api shutting down")
	return s.httpServer.Shutdown(ctx)
}
