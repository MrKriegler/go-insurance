package core

import (
	"context"
	"fmt"
	"time"
)

type PolicyStatus string

const (
	PolicyStatusActive    PolicyStatus = "active"
	PolicyStatusLapsed    PolicyStatus = "lapsed"
	PolicyStatusCancelled PolicyStatus = "cancelled"
	PolicyStatusExpired   PolicyStatus = "expired"
)

// Policy represents an issued insurance policy.
type Policy struct {
	ID             string       `json:"id"`
	Number         string       `json:"number"` // Human-readable policy number (e.g., POL-2025-000001)
	ApplicationID  string       `json:"application_id"`
	OfferID        string       `json:"offer_id"`
	ProductSlug    string       `json:"product_slug"`
	CoverageAmount int64        `json:"coverage_amount"`
	TermYears      int          `json:"term_years"`
	MonthlyPremium float64      `json:"monthly_premium"`
	Insured        Applicant    `json:"insured"` // Snapshot of applicant at issuance
	Status         PolicyStatus `json:"status"`
	EffectiveDate  time.Time    `json:"effective_date"` // When coverage begins
	ExpiryDate     time.Time    `json:"expiry_date"` // EffectiveDate + TermYears
	IssuedAt       time.Time    `json:"issued_at"`
}

type PolicyFilter struct {
	ApplicationID string
	Status        PolicyStatus
}

type PolicyRepo interface {
	Create(ctx context.Context, policy Policy) error
	Get(ctx context.Context, id string) (Policy, error)
	GetByNumber(ctx context.Context, number string) (Policy, error)
	GetByOfferID(ctx context.Context, offerID string) (Policy, error)
	GetByApplicationID(ctx context.Context, appID string) (Policy, error)
	List(ctx context.Context, filter PolicyFilter, limit, offset int) ([]Policy, int64, error)
	NextPolicyNumber(ctx context.Context) (string, error)
}

var (
	ErrPolicyNotFound = fmt.Errorf("%w: policy not found", ErrNotFound)
	ErrPolicyExists   = fmt.Errorf("%w: policy already exists for offer", ErrConflict)
)
