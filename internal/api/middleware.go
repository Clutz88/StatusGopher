package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Clutz88/StatusGopher/internal/logging"
	"github.com/google/uuid"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(rr, r)

		logging.FromCtx(r.Context()).Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rr.statusCode,
			"duration", time.Since(start),
		)
	})
}

func traceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		logger := slog.Default().With("trace_id", id)
		ctx := context.WithValue(r.Context(), logging.LoggerKey{}, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
