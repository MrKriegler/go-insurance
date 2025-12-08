package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/platform/ids"
)

type PolicyService interface {
	// IssueFromOffer creates a policy from an accepted offer (called by issuance worker)
	IssueFromOffer(ctx context.Context, offerID string) (Policy, error)

	// Get retrieves a policy by ID
	Get(ctx context.Context, id string) (Policy, error)

	// GetByNumber retrieves a policy by policy number
	GetByNumber(ctx context.Context, number string) (Policy, error)

	// List returns policies with optional filtering and pagination
	List(ctx context.Context, filter PolicyFilter, limit, offset int) ([]Policy, int64, error)
}

type policyService struct {
	policies PolicyRepo
	offers   OfferRepo
	apps     ApplicationRepo
	clock    func() time.Time
}

func NewPolicyService(policies PolicyRepo, offers OfferRepo, apps ApplicationRepo) PolicyService {
	return &policyService{
		policies: policies,
		offers:   offers,
		apps:     apps,
		clock:    time.Now,
	}
}

func (s *policyService) IssueFromOffer(ctx context.Context, offerID string) (Policy, error) {
	// 1) Load offer
	offer, err := s.offers.Get(ctx, offerID)
	if err != nil {
		return Policy{}, err
	}

	// 2) Verify offer is accepted
	if offer.Status != OfferStatusAccepted {
		return Policy{}, ErrOfferNotAccepted
	}

	// 3) Check if policy already exists for this offer
	existing, err := s.policies.GetByOfferID(ctx, offerID)
	if err == nil {
		// Policy already exists
		return existing, nil
	}
	if !errors.Is(err, ErrPolicyNotFound) {
		return Policy{}, err
	}

	// 4) Load application for insured details
	app, err := s.apps.Get(ctx, offer.ApplicationID)
	if err != nil {
		return Policy{}, err
	}

	// 5) Generate policy number
	policyNumber, err := s.policies.NextPolicyNumber(ctx)
	if err != nil {
		return Policy{}, fmt.Errorf("failed to generate policy number: %w", err)
	}

	// 6) Calculate dates
	now := s.clock()
	effectiveDate := now
	expiryDate := effectiveDate.AddDate(offer.TermYears, 0, 0)

	// 7) Create policy
	policy := Policy{
		ID:             ids.New(),
		Number:         policyNumber,
		ApplicationID:  offer.ApplicationID,
		OfferID:        offer.ID,
		ProductSlug:    offer.ProductSlug,
		CoverageAmount: offer.CoverageAmount,
		TermYears:      offer.TermYears,
		MonthlyPremium: offer.MonthlyPremium,
		Insured:        app.Applicant,
		Status:         PolicyStatusActive,
		EffectiveDate:  effectiveDate,
		ExpiryDate:     expiryDate,
		IssuedAt:       now,
	}

	// 8) Save policy
	if err := s.policies.Create(ctx, policy); err != nil {
		if errors.Is(err, ErrPolicyExists) {
			// Race condition - policy was created by another process
			return s.policies.GetByOfferID(ctx, offerID)
		}
		return Policy{}, err
	}

	// 9) Update offer status to issued (best-effort, policy is already created)
	offer.Status = OfferStatusIssued
	_ = s.offers.Update(ctx, offer) // Ignore error - policy exists, offer update is non-critical

	return policy, nil
}

func (s *policyService) Get(ctx context.Context, id string) (Policy, error) {
	if id == "" {
		return Policy{}, fmt.Errorf("%w: missing policy ID", ErrValidation)
	}
	return s.policies.Get(ctx, id)
}

func (s *policyService) GetByNumber(ctx context.Context, number string) (Policy, error) {
	if number == "" {
		return Policy{}, fmt.Errorf("%w: missing policy number", ErrValidation)
	}
	return s.policies.GetByNumber(ctx, number)
}

func (s *policyService) List(ctx context.Context, filter PolicyFilter, limit, offset int) ([]Policy, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.policies.List(ctx, filter, limit, offset)
}
