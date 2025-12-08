package mongo

import (
	"time"

	"github.com/MrKriegler/go-insurance/internal/core"
)

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

// Quote
type QuoteDoc struct {
	ID             string    `bson:"_id"`
	ProductID      string    `bson:"product_id"`
	ProductSlug    string    `bson:"product_slug"`
	CoverageAmount int64     `bson:"coverage_amount"`
	TermYears      int       `bson:"term_years"`
	MonthlyPremium float64   `bson:"monthly_premium"`
	Status         string    `bson:"status"`
	CreatedAt      time.Time `bson:"created_at"`
	ExpiresAt      time.Time `bson:"expires_at"`
}

func fromQuoteDoc(d QuoteDoc) core.Quote {
	return core.Quote{
		ID:             d.ID,
		ProductID:      d.ProductID,
		ProductSlug:    d.ProductSlug,
		CoverageAmount: d.CoverageAmount,
		TermYears:      d.TermYears,
		MonthlyPremium: d.MonthlyPremium,
		Status:         core.QuoteStatus(d.Status),
		CreatedAt:      d.CreatedAt,
		ExpiresAt:      d.ExpiresAt,
	}
}

func toQuoteDoc(q core.Quote) QuoteDoc {
	return QuoteDoc{
		ID:             q.ID,
		ProductID:      q.ProductID,
		ProductSlug:    q.ProductSlug,
		CoverageAmount: q.CoverageAmount,
		TermYears:      q.TermYears,
		MonthlyPremium: q.MonthlyPremium,
		Status:         string(q.Status),
		CreatedAt:      q.CreatedAt,
		ExpiresAt:      q.ExpiresAt,
	}
}

// Applicant
type ApplicantDoc struct {
	FirstName   string `bson:"first_name"`
	LastName    string `bson:"last_name"`
	Email       string `bson:"email"`
	DateOfBirth string `bson:"date_of_birth"`
	Age         int    `bson:"age"`
	Smoker      bool   `bson:"smoker"`
	State       string `bson:"state"`
}

func fromApplicantDoc(d ApplicantDoc) core.Applicant {
	return core.Applicant{
		FirstName:   d.FirstName,
		LastName:    d.LastName,
		Email:       d.Email,
		DateOfBirth: d.DateOfBirth,
		Age:         d.Age,
		Smoker:      d.Smoker,
		State:       d.State,
	}
}

func toApplicantDoc(a core.Applicant) ApplicantDoc {
	return ApplicantDoc{
		FirstName:   a.FirstName,
		LastName:    a.LastName,
		Email:       a.Email,
		DateOfBirth: a.DateOfBirth,
		Age:         a.Age,
		Smoker:      a.Smoker,
		State:       a.State,
	}
}

// Application
type ApplicationDoc struct {
	ID             string       `bson:"_id"`
	QuoteID        string       `bson:"quote_id"`
	ProductID      string       `bson:"product_id"`
	ProductSlug    string       `bson:"product_slug"`
	CoverageAmount int64        `bson:"coverage_amount"`
	TermYears      int          `bson:"term_years"`
	MonthlyPremium float64      `bson:"monthly_premium"`
	Applicant      ApplicantDoc `bson:"applicant"`
	Status         string       `bson:"status"`
	CreatedAt      time.Time    `bson:"created_at"`
	UpdatedAt      time.Time    `bson:"updated_at"`
	SubmittedAt    *time.Time   `bson:"submitted_at,omitempty"`
}

func fromApplicationDoc(d ApplicationDoc) core.Application {
	return core.Application{
		ID:             d.ID,
		QuoteID:        d.QuoteID,
		ProductID:      d.ProductID,
		ProductSlug:    d.ProductSlug,
		CoverageAmount: d.CoverageAmount,
		TermYears:      d.TermYears,
		MonthlyPremium: d.MonthlyPremium,
		Applicant:      fromApplicantDoc(d.Applicant),
		Status:         core.ApplicationStatus(d.Status),
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
		SubmittedAt:    d.SubmittedAt,
	}
}

func toApplicationDoc(a core.Application) ApplicationDoc {
	return ApplicationDoc{
		ID:             a.ID,
		QuoteID:        a.QuoteID,
		ProductID:      a.ProductID,
		ProductSlug:    a.ProductSlug,
		CoverageAmount: a.CoverageAmount,
		TermYears:      a.TermYears,
		MonthlyPremium: a.MonthlyPremium,
		Applicant:      toApplicantDoc(a.Applicant),
		Status:         string(a.Status),
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
		SubmittedAt:    a.SubmittedAt,
	}
}

// UnderwritingCase
type RiskFactorsDoc struct {
	Age            int   `bson:"age"`
	Smoker         bool  `bson:"smoker"`
	CoverageAmount int64 `bson:"coverage_amount"`
	TermYears      int   `bson:"term_years"`
}

type RiskScoreDoc struct {
	Score       int      `bson:"score"`
	Flags       []string `bson:"flags"`
	Recommended string   `bson:"recommended"`
}

type UnderwritingCaseDoc struct {
	ID            string         `bson:"_id"`
	ApplicationID string         `bson:"application_id"`
	RiskFactors   RiskFactorsDoc `bson:"risk_factors"`
	RiskScore     RiskScoreDoc   `bson:"risk_score"`
	Decision      string         `bson:"decision"`
	Method        string         `bson:"method"`
	DecidedBy     string         `bson:"decided_by"`
	Reason        string         `bson:"reason"`
	CreatedAt     time.Time      `bson:"created_at"`
	UpdatedAt     time.Time      `bson:"updated_at"`
	DecidedAt     *time.Time     `bson:"decided_at,omitempty"`
}

func fromUnderwritingCaseDoc(d UnderwritingCaseDoc) core.UnderwritingCase {
	return core.UnderwritingCase{
		ID:            d.ID,
		ApplicationID: d.ApplicationID,
		RiskFactors: core.RiskFactors{
			Age:            d.RiskFactors.Age,
			Smoker:         d.RiskFactors.Smoker,
			CoverageAmount: d.RiskFactors.CoverageAmount,
			TermYears:      d.RiskFactors.TermYears,
		},
		RiskScore: core.RiskScore{
			Score:       d.RiskScore.Score,
			Flags:       d.RiskScore.Flags,
			Recommended: core.UWDecision(d.RiskScore.Recommended),
		},
		Decision:  core.UWDecision(d.Decision),
		Method:    core.UWMethod(d.Method),
		DecidedBy: d.DecidedBy,
		Reason:    d.Reason,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		DecidedAt: d.DecidedAt,
	}
}

func toUnderwritingCaseDoc(uw core.UnderwritingCase) UnderwritingCaseDoc {
	return UnderwritingCaseDoc{
		ID:            uw.ID,
		ApplicationID: uw.ApplicationID,
		RiskFactors: RiskFactorsDoc{
			Age:            uw.RiskFactors.Age,
			Smoker:         uw.RiskFactors.Smoker,
			CoverageAmount: uw.RiskFactors.CoverageAmount,
			TermYears:      uw.RiskFactors.TermYears,
		},
		RiskScore: RiskScoreDoc{
			Score:       uw.RiskScore.Score,
			Flags:       uw.RiskScore.Flags,
			Recommended: string(uw.RiskScore.Recommended),
		},
		Decision:  string(uw.Decision),
		Method:    string(uw.Method),
		DecidedBy: uw.DecidedBy,
		Reason:    uw.Reason,
		CreatedAt: uw.CreatedAt,
		UpdatedAt: uw.UpdatedAt,
		DecidedAt: uw.DecidedAt,
	}
}

// Offer
type OfferDoc struct {
	ID             string     `bson:"_id"`
	ApplicationID  string     `bson:"application_id"`
	ProductSlug    string     `bson:"product_slug"`
	CoverageAmount int64      `bson:"coverage_amount"`
	TermYears      int        `bson:"term_years"`
	MonthlyPremium float64    `bson:"monthly_premium"`
	Status         string     `bson:"status"`
	CreatedAt      time.Time  `bson:"created_at"`
	ExpiresAt      time.Time  `bson:"expires_at"`
	AcceptedAt     *time.Time `bson:"accepted_at,omitempty"`
	DeclinedAt     *time.Time `bson:"declined_at,omitempty"`
}

func fromOfferDoc(d OfferDoc) core.Offer {
	return core.Offer{
		ID:             d.ID,
		ApplicationID:  d.ApplicationID,
		ProductSlug:    d.ProductSlug,
		CoverageAmount: d.CoverageAmount,
		TermYears:      d.TermYears,
		MonthlyPremium: d.MonthlyPremium,
		Status:         core.OfferStatus(d.Status),
		CreatedAt:      d.CreatedAt,
		ExpiresAt:      d.ExpiresAt,
		AcceptedAt:     d.AcceptedAt,
		DeclinedAt:     d.DeclinedAt,
	}
}

func toOfferDoc(o core.Offer) OfferDoc {
	return OfferDoc{
		ID:             o.ID,
		ApplicationID:  o.ApplicationID,
		ProductSlug:    o.ProductSlug,
		CoverageAmount: o.CoverageAmount,
		TermYears:      o.TermYears,
		MonthlyPremium: o.MonthlyPremium,
		Status:         string(o.Status),
		CreatedAt:      o.CreatedAt,
		ExpiresAt:      o.ExpiresAt,
		AcceptedAt:     o.AcceptedAt,
		DeclinedAt:     o.DeclinedAt,
	}
}

// Policy
type PolicyDoc struct {
	ID             string       `bson:"_id"`
	Number         string       `bson:"number"`
	ApplicationID  string       `bson:"application_id"`
	OfferID        string       `bson:"offer_id"`
	ProductSlug    string       `bson:"product_slug"`
	CoverageAmount int64        `bson:"coverage_amount"`
	TermYears      int          `bson:"term_years"`
	MonthlyPremium float64      `bson:"monthly_premium"`
	Insured        ApplicantDoc `bson:"insured"`
	Status         string       `bson:"status"`
	EffectiveDate  time.Time    `bson:"effective_date"`
	ExpiryDate     time.Time    `bson:"expiry_date"`
	IssuedAt       time.Time    `bson:"issued_at"`
}

func fromPolicyDoc(d PolicyDoc) core.Policy {
	return core.Policy{
		ID:             d.ID,
		Number:         d.Number,
		ApplicationID:  d.ApplicationID,
		OfferID:        d.OfferID,
		ProductSlug:    d.ProductSlug,
		CoverageAmount: d.CoverageAmount,
		TermYears:      d.TermYears,
		MonthlyPremium: d.MonthlyPremium,
		Insured:        fromApplicantDoc(d.Insured),
		Status:         core.PolicyStatus(d.Status),
		EffectiveDate:  d.EffectiveDate,
		ExpiryDate:     d.ExpiryDate,
		IssuedAt:       d.IssuedAt,
	}
}

func toPolicyDoc(p core.Policy) PolicyDoc {
	return PolicyDoc{
		ID:             p.ID,
		Number:         p.Number,
		ApplicationID:  p.ApplicationID,
		OfferID:        p.OfferID,
		ProductSlug:    p.ProductSlug,
		CoverageAmount: p.CoverageAmount,
		TermYears:      p.TermYears,
		MonthlyPremium: p.MonthlyPremium,
		Insured:        toApplicantDoc(p.Insured),
		Status:         string(p.Status),
		EffectiveDate:  p.EffectiveDate,
		ExpiryDate:     p.ExpiryDate,
		IssuedAt:       p.IssuedAt,
	}
}
