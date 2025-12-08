package core

import (
	"context"
	"fmt"
	"time"
)

type QuoteStatus string

const (
	QuoteStatusNew     QuoteStatus = "new"
	QuoteStatusPriced  QuoteStatus = "priced"
	QuoteStatusExpired QuoteStatus = "expired"
)

type QuoteInput struct {
	ProductSlug    string `json:"product_slug"`
	CoverageAmount int64  `json:"coverage_amount"`
	TermYears      int    `json:"term_years"`

	Age    int  `json:"age"`
	Smoker bool `json:"smoker"`
}

type Quote struct {
	ID             string      `json:"id"`
	ProductID      string      `json:"product_id"`
	ProductSlug    string      `json:"product_slug"`
	CoverageAmount int64       `json:"coverage_amount"`
	TermYears      int         `json:"term_years"`
	MonthlyPremium float64     `json:"monthly_premium"`
	Status         QuoteStatus `json:"status"`
	CreatedAt      time.Time   `json:"created_at"`
	ExpiresAt      time.Time   `json:"expires_at"`
}

type QuoteRepo interface {
	Create(ctx context.Context, q Quote) error
	Get(ctx context.Context, id string) (Quote, error)
}

// Pricing is pure domain/service logic; no I/O beyond reading product(s).
type QuoteService interface {
	Price(ctx context.Context, in QuoteInput) (Quote, error)
}

func (in QuoteInput) Validate() error {
	if in.ProductSlug == "" {
		return fmt.Errorf("%w: missing product slug", ErrValidation)
	}
	if in.CoverageAmount <= 0 {
		return fmt.Errorf("%w: coverage must be > 0", ErrValidation)
	}
	if in.TermYears <= 0 {
		return fmt.Errorf("%w: term must be > 0", ErrValidation)
	}
	if in.Age <= 0 || in.Age > 120 {
		return fmt.Errorf("%w: invalid age", ErrValidation)
	}
	return nil
}

var (
	ErrQuoteNotFound = fmt.Errorf("%w: quote not found", ErrNotFound)
)
