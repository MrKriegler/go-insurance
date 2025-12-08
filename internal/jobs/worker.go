package jobs

import (
	"context"
	"log/slog"
	"time"
)

// Worker defines a background job that polls for work.
type Worker interface {
	Start(ctx context.Context)
	Name() string
}

// BaseWorker provides common polling infrastructure.
type BaseWorker struct {
	name     string
	interval time.Duration
	log      *slog.Logger
}

// NewBaseWorker creates a new base worker.
func NewBaseWorker(name string, interval time.Duration, log *slog.Logger) BaseWorker {
	return BaseWorker{
		name:     name,
		interval: interval,
		log:      log.With("worker", name),
	}
}

// Poll runs the work function at regular intervals until context is cancelled.
func (w *BaseWorker) Poll(ctx context.Context, work func(context.Context) error) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.log.Info("worker started", "interval", w.interval)

	// Run immediately on start
	if err := work(ctx); err != nil {
		w.log.Error("worker error", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker stopping")
			return
		case <-ticker.C:
			if err := work(ctx); err != nil {
				w.log.Error("worker error", "err", err)
			}
		}
	}
}
