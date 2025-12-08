package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/platform/ids"
)

type quoteService struct {
	products ProductRepo
	quotes   QuoteRepo
	clock    func() time.Time
}

func NewQuoteService(products ProductRepo, quotes QuoteRepo) QuoteService {
	return &quoteService{
		products: products,
		quotes:   quotes,
		clock:    time.Now,
	}
}

func (s *quoteService) Price(ctx context.Context, in QuoteInput) (Quote, error) {
	// 1) validate inputs
	if err := in.Validate(); err != nil {
		return Quote{}, err
	}

	// 2) load product by slug
	p, err := s.products.GetBySlug(ctx, in.ProductSlug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Quote{}, fmt.Errorf("%w: product %q", ErrNotFound, in.ProductSlug)
		}
		return Quote{}, err
	}

	// 3) validation against product bounds
	if in.CoverageAmount < p.MinCoverage || in.CoverageAmount > p.MaxCoverage {
		return Quote{}, fmt.Errorf("%w: coverage must be between %d and %d",
			ErrValidation, p.MinCoverage, p.MaxCoverage)
	}
	if in.TermYears != p.TermYears {
		return Quote{}, fmt.Errorf("%w: term must be %d years for product %s",
			ErrValidation, p.TermYears, p.Slug)
	}

	// 4) price
	base := float64(in.CoverageAmount) / 1000.0 * p.BaseRate
	ageFactor := factorAge(in.Age)
	smokerFactor := factorSmoker(in.Smoker)

	price := base * ageFactor * smokerFactor

	now := s.clock()
	q := Quote{
		ID:             ids.New(),
		ProductID:      p.ID,
		ProductSlug:    p.Slug,
		CoverageAmount: in.CoverageAmount,
		TermYears:      in.TermYears,
		MonthlyPremium: round2(price),
		Status:         QuoteStatusPriced,
		CreatedAt:      now,
		ExpiresAt:      now.Add(24 * time.Hour), // simple: quote valid for 1 day
	}

	// 5) persist
	if s.quotes != nil {
		if err := s.quotes.Create(ctx, q); err != nil {
			return Quote{}, err
		}
	}
	return q, nil
}

func factorAge(age int) float64 {
	switch {
	case age <= 30:
		return 0.90
	case age <= 40:
		return 1.00
	case age <= 50:
		return 1.20
	case age <= 60:
		return 1.60
	default:
		return 2.00
	}
}

func factorSmoker(smoker bool) float64 {
	if smoker {
		return 1.50
	}
	return 1.00
}

func round2(x float64) float64 {
	return float64(int64(x*100+0.5)) / 100.0
}
