package mongo

import "github.com/MrKriegler/go-insurance/internal/core"

const (
	ColProducts     = "products"
	ColQuotes       = "quotes"
	ColApplications = "applications"
	ColUnderwriting = "underwriting_cases"
	ColOffers       = "offers"
	ColPolicies     = "policies"
)

// Product
type ProductDoc struct {
	ID          string  `bson:"_id"`
	Slug        string  `bson:"slug"` // unique index
	Name        string  `bson:"name"`
	TermYears   int     `bson:"term_years"`
	MinCoverage int64   `bson:"min_coverage"`
	MaxCoverage int64   `bson:"max_coverage"`
	BaseRate    float64 `bson:"base_rate"`
}

func fromProductDoc(d ProductDoc) core.Product {
	return core.Product{
		ID:          d.ID,
		Slug:        d.Slug,
		Name:        d.Name,
		TermYears:   d.TermYears,
		MinCoverage: d.MinCoverage,
		MaxCoverage: d.MaxCoverage,
		BaseRate:    d.BaseRate,
	}
}

func toProductDoc(p core.Product) ProductDoc {
	return ProductDoc{
		ID:          p.ID,
		Slug:        p.Slug,
		Name:        p.Name,
		TermYears:   p.TermYears,
		MinCoverage: p.MinCoverage,
		MaxCoverage: p.MaxCoverage,
		BaseRate:    p.BaseRate,
	}
}
