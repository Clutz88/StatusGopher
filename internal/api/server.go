package api

import (
	"context"
	"log"
	"net/http"

	"github.com/Clutz88/StatusGopher/internal/database"
)

type Server struct {
	db         *database.DB
	httpServer *http.Server
}

func NewServer(addr string, db *database.DB) *Server {
	s := &Server{db: db}

	mux := http.NewServeMux()
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s
}

func (s *Server) Start() error {
	log.Printf("API server listening on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Println("API server shutting down...")
	return s.httpServer.Shutdown(ctx)
}
