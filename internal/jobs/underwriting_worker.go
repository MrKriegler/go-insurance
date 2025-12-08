package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
)

// UnderwritingWorker processes submitted applications through underwriting.
type UnderwritingWorker struct {
	BaseWorker
	apps core.ApplicationRepo
	uw   core.UnderwritingService
}

// NewUnderwritingWorker creates a new underwriting worker.
func NewUnderwritingWorker(
	apps core.ApplicationRepo,
	uwSvc core.UnderwritingService,
	interval time.Duration,
	log *slog.Logger,
) *UnderwritingWorker {
	return &UnderwritingWorker{
		BaseWorker: NewBaseWorker("underwriting", interval, log),
		apps:       apps,
		uw:         uwSvc,
	}
}

// Start begins the worker polling loop.
func (w *UnderwritingWorker) Start(ctx context.Context) {
	w.Poll(ctx, w.processSubmitted)
}

// Name returns the worker name.
func (w *UnderwritingWorker) Name() string {
	return w.name
}

// processSubmitted finds and processes submitted applications.
func (w *UnderwritingWorker) processSubmitted(ctx context.Context) error {
	// Find applications in "submitted" status (limit 10 per poll)
	apps, err := w.apps.FindByStatus(ctx, core.ApplicationStatusSubmitted, 10)
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		return nil
	}

	w.log.Info("found submitted applications", "count", len(apps))

	for _, app := range apps {
		w.log.Info("processing application", "app_id", app.ID)

		// Run underwriting for each
		uwCase, err := w.uw.ProcessApplication(ctx, app.ID)
		if err != nil {
			w.log.Error("failed to process application",
				"app_id", app.ID,
				"err", err,
			)
			continue
		}

		w.log.Info("underwriting complete",
			"app_id", app.ID,
			"case_id", uwCase.ID,
			"decision", uwCase.Decision,
			"method", uwCase.Method,
		)
	}

	return nil
}
