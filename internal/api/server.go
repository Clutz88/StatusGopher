package api

import (
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
