package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/platform/ids"
)

type OfferService interface {
	// GenerateOffer creates an offer from an approved application
	GenerateOffer(ctx context.Context, appID string) (Offer, error)

	// Get retrieves an offer by ID
	Get(ctx context.Context, id string) (Offer, error)

	// GetByApplicationID retrieves an offer by application ID
	GetByApplicationID(ctx context.Context, appID string) (Offer, error)

	// Accept marks an offer as accepted
	Accept(ctx context.Context, id string) (Offer, error)

	// Decline marks an offer as declined
	Decline(ctx context.Context, id string) (Offer, error)
}

type offerService struct {
	offers OfferRepo
	apps   ApplicationRepo
	clock  func() time.Time
}

func NewOfferService(offers OfferRepo, apps ApplicationRepo) OfferService {
	return &offerService{
		offers: offers,
		apps:   apps,
		clock:  time.Now,
	}
}

func (s *offerService) GenerateOffer(ctx context.Context, appID string) (Offer, error) {
	// 1) Load application
	app, err := s.apps.Get(ctx, appID)
	if err != nil {
		return Offer{}, err
	}

	// 2) Verify application is approved
	if app.Status != ApplicationStatusApproved {
		return Offer{}, ErrAppNotApproved
	}

	// 3) Check if offer already exists
	existing, err := s.offers.GetByApplicationID(ctx, appID)
	if err == nil {
		// Offer already exists
		return existing, nil
	}
	if !errors.Is(err, ErrOfferNotFound) {
		return Offer{}, err
	}

	// 4) Create offer
	now := s.clock()
	offer := Offer{
		ID:             ids.New(),
		ApplicationID:  app.ID,
		ProductSlug:    app.ProductSlug,
		CoverageAmount: app.CoverageAmount,
		TermYears:      app.TermYears,
		MonthlyPremium: app.MonthlyPremium,
		Status:         OfferStatusPending,
		CreatedAt:      now,
		ExpiresAt:      now.AddDate(0, 0, OfferValidityDays),
	}

	// 5) Persist
	if err := s.offers.Create(ctx, offer); err != nil {
		if errors.Is(err, ErrOfferExists) {
			// Race condition - offer was created by another process
			return s.offers.GetByApplicationID(ctx, appID)
		}
		return Offer{}, err
	}

	return offer, nil
}

func (s *offerService) Get(ctx context.Context, id string) (Offer, error) {
	if id == "" {
		return Offer{}, fmt.Errorf("%w: missing offer ID", ErrValidation)
	}
	return s.offers.Get(ctx, id)
}

func (s *offerService) GetByApplicationID(ctx context.Context, appID string) (Offer, error) {
	if appID == "" {
		return Offer{}, fmt.Errorf("%w: missing application ID", ErrValidation)
	}
	return s.offers.GetByApplicationID(ctx, appID)
}

func (s *offerService) Accept(ctx context.Context, id string) (Offer, error) {
	// 1) Load offer
	offer, err := s.offers.Get(ctx, id)
	if err != nil {
		return Offer{}, err
	}

	// 2) Verify offer is pending
	if offer.Status != OfferStatusPending {
		return Offer{}, ErrOfferNotPending
	}

	// 3) Check if expired
	now := s.clock()
	if offer.IsExpired(now) {
		// Update status to expired (best-effort)
		offer.Status = OfferStatusExpired
		offer.AcceptedAt = nil
		_ = s.offers.Update(ctx, offer) // Ignore error - returning ErrOfferExpired anyway
		return Offer{}, ErrOfferExpired
	}

	// 4) Update offer
	offer.Status = OfferStatusAccepted
	offer.AcceptedAt = &now

	if err := s.offers.Update(ctx, offer); err != nil {
		return Offer{}, err
	}

	return offer, nil
}

func (s *offerService) Decline(ctx context.Context, id string) (Offer, error) {
	// 1) Load offer
	offer, err := s.offers.Get(ctx, id)
	if err != nil {
		return Offer{}, err
	}

	// 2) Verify offer is pending
	if offer.Status != OfferStatusPending {
		return Offer{}, ErrOfferNotPending
	}

	// 3) Update offer
	now := s.clock()
	offer.Status = OfferStatusDeclined
	offer.DeclinedAt = &now

	if err := s.offers.Update(ctx, offer); err != nil {
		return Offer{}, err
	}

	return offer, nil
}
