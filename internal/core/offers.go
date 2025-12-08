package core

import (
	"context"
	"fmt"
	"time"
)

type OfferStatus string

const (
	OfferStatusPending  OfferStatus = "pending"
	OfferStatusAccepted OfferStatus = "accepted"
	OfferStatusDeclined OfferStatus = "declined"
	OfferStatusExpired  OfferStatus = "expired"
	OfferStatusIssued   OfferStatus = "issued" // Policy has been issued
)

const (
	// OfferValidityDays is how long an offer remains valid.
	OfferValidityDays = 30
)

// Offer represents the terms offered to an approved applicant.
type Offer struct {
	ID             string      `json:"id"`
	ApplicationID  string      `json:"application_id"`
	ProductSlug    string      `json:"product_slug"`
	CoverageAmount int64       `json:"coverage_amount"`
	TermYears      int         `json:"term_years"`
	MonthlyPremium float64     `json:"monthly_premium"`
	Status         OfferStatus `json:"status"`
	CreatedAt      time.Time   `json:"created_at"`
	ExpiresAt      time.Time   `json:"expires_at"`
	AcceptedAt     *time.Time  `json:"accepted_at,omitempty"`
	DeclinedAt     *time.Time  `json:"declined_at,omitempty"`
}

type OfferRepo interface {
	Create(ctx context.Context, offer Offer) error
	Get(ctx context.Context, id string) (Offer, error)
	GetByApplicationID(ctx context.Context, appID string) (Offer, error)
	Update(ctx context.Context, offer Offer) error
	FindAccepted(ctx context.Context, limit int) ([]Offer, error)
	ExpireOffers(ctx context.Context, before time.Time) (int64, error)
}

// CanTransitionTo checks if a status transition is valid.
func (s OfferStatus) CanTransitionTo(next OfferStatus) bool {
	transitions := map[OfferStatus][]OfferStatus{
		OfferStatusPending:  {OfferStatusAccepted, OfferStatusDeclined, OfferStatusExpired},
		OfferStatusAccepted: {OfferStatusIssued},
	}
	for _, allowed := range transitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// IsExpired checks if the offer has expired.
func (o Offer) IsExpired(now time.Time) bool {
	return now.After(o.ExpiresAt)
}

var (
	ErrOfferNotFound     = fmt.Errorf("%w: offer not found", ErrNotFound)
	ErrOfferExists       = fmt.Errorf("%w: offer already exists for application", ErrConflict)
	ErrOfferExpired      = fmt.Errorf("%w: offer has expired", ErrInvalidState)
	ErrOfferNotPending   = fmt.Errorf("%w: offer is not in pending status", ErrInvalidState)
	ErrOfferNotAccepted  = fmt.Errorf("%w: offer is not in accepted status", ErrInvalidState)
	ErrAppNotApproved    = fmt.Errorf("%w: application is not approved", ErrInvalidState)
)
