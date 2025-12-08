package core

import (
	"context"
	"fmt"
)

type Product struct {
	ID          string  `json:"id"`
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	TermYears   int     `json:"term_years"`
	MinCoverage int64   `json:"min_coverage"`
	MaxCoverage int64   `json:"max_coverage"`
	BaseRate    float64 `json:"base_rate"` // Base monthly rate per 1,000 units of coverage
}

type ProductRepo interface {
	List(ctx context.Context) ([]Product, error)
	GetBySlug(ctx context.Context, slug string) (Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpsertBySlug(ctx context.Context, p Product) error
}

func (p Product) Validate() error {
	if p.TermYears <= 0 {
		return fmt.Errorf("%v: term must be > 0", ErrValidation)
	}
	if p.MinCoverage <= 0 || p.MaxCoverage < p.MinCoverage {
		return fmt.Errorf("%v: invalid coverage range", ErrValidation)
	}
	if p.BaseRate <= 0 {
		return fmt.Errorf("%v: base rate must be > 0", ErrValidation)
	}
	if p.Name == "" {
		return fmt.Errorf("%v: missing name", ErrValidation)
	}
	return nil
}

// Error helpers pertaining to products.
var (
	ErrProductNotFound = fmt.Errorf("%v: product not found", ErrNotFound)
	ErrProductConflict = fmt.Errorf("%v: product already exists", ErrConflict)
)
