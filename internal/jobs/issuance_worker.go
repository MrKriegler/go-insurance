package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
)

// IssuanceWorker processes accepted offers and issues policies.
type IssuanceWorker struct {
	BaseWorker
	offers   core.OfferRepo
	policies core.PolicyService
}

// NewIssuanceWorker creates a new issuance worker.
func NewIssuanceWorker(
	offers core.OfferRepo,
	policySvc core.PolicyService,
	interval time.Duration,
	log *slog.Logger,
) *IssuanceWorker {
	return &IssuanceWorker{
		BaseWorker: NewBaseWorker("issuance", interval, log),
		offers:     offers,
		policies:   policySvc,
	}
}

// Start begins the worker polling loop.
func (w *IssuanceWorker) Start(ctx context.Context) {
	w.Poll(ctx, w.processAccepted)
}

// Name returns the worker name.
func (w *IssuanceWorker) Name() string {
	return w.name
}

// processAccepted finds and processes accepted offers.
func (w *IssuanceWorker) processAccepted(ctx context.Context) error {
	// Find offers in "accepted" status (limit 10 per poll)
	offers, err := w.offers.FindAccepted(ctx, 10)
	if err != nil {
		return err
	}

	if len(offers) == 0 {
		return nil
	}

	w.log.Info("found accepted offers", "count", len(offers))

	for _, offer := range offers {
		w.log.Info("issuing policy", "offer_id", offer.ID)

		// Issue policy for each
		policy, err := w.policies.IssueFromOffer(ctx, offer.ID)
		if err != nil {
			w.log.Error("failed to issue policy",
				"offer_id", offer.ID,
				"err", err,
			)
			continue
		}

		w.log.Info("policy issued",
			"offer_id", offer.ID,
			"policy_id", policy.ID,
			"policy_number", policy.Number,
		)
	}

	return nil
}
