package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

// New builds a health check HTTP handler with liveness and readiness endpoints.
func New(log *slog.Logger, p Pinger, opTimeout time.Duration) http.Handler {
	r := chi.NewRouter()

	// Liveness: process is up
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Readiness: dependencies are reachable
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), opTimeout)
		defer cancel()

		if err := p.Ping(ctx); err != nil {
			if log != nil {
				log.Warn("readiness failed", "err", err)
			}
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	return r
}
