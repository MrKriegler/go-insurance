package core

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

type ApplicationStatus string

const (
	ApplicationStatusDraft       ApplicationStatus = "draft"
	ApplicationStatusSubmitted   ApplicationStatus = "submitted"
	ApplicationStatusUnderReview ApplicationStatus = "under_review"
	ApplicationStatusApproved    ApplicationStatus = "approved"
	ApplicationStatusDeclined    ApplicationStatus = "declined"
)

// Applicant contains personal information for underwriting.
type Applicant struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	DateOfBirth string `json:"date_of_birth"` // YYYY-MM-DD format
	Age         int    `json:"age"`
	Smoker      bool   `json:"smoker"`
	State       string `json:"state"` // US state code
}

// Application links a quote to applicant information.
type Application struct {
	ID             string            `json:"id"`
	QuoteID        string            `json:"quote_id"`
	ProductID      string            `json:"product_id"`
	ProductSlug    string            `json:"product_slug"`
	CoverageAmount int64             `json:"coverage_amount"`
	TermYears      int               `json:"term_years"`
	MonthlyPremium float64           `json:"monthly_premium"`
	Applicant      Applicant         `json:"applicant"`
	Status         ApplicationStatus `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	SubmittedAt    *time.Time        `json:"submitted_at,omitempty"`
}

type ApplicationInput struct {
	QuoteID   string    `json:"quote_id"`
	Applicant Applicant `json:"applicant"`
}

type ApplicationPatch struct {
	Applicant *Applicant `json:"applicant,omitempty"`
}

type ApplicationRepo interface {
	Create(ctx context.Context, app Application) error
	Get(ctx context.Context, id string) (Application, error)
	Update(ctx context.Context, app Application) error
	UpdateStatus(ctx context.Context, id string, status ApplicationStatus, updatedAt time.Time) error
	FindByStatus(ctx context.Context, status ApplicationStatus, limit int) ([]Application, error)
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (a Applicant) Validate() error {
	if a.FirstName == "" {
		return fmt.Errorf("%w: first name is required", ErrValidation)
	}
	if a.LastName == "" {
		return fmt.Errorf("%w: last name is required", ErrValidation)
	}
	if a.Email == "" {
		return fmt.Errorf("%w: email is required", ErrValidation)
	}
	if !emailRegex.MatchString(a.Email) {
		return fmt.Errorf("%w: invalid email format", ErrValidation)
	}
	if a.DateOfBirth == "" {
		return fmt.Errorf("%w: date of birth is required", ErrValidation)
	}
	if a.Age < 18 || a.Age > 120 {
		return fmt.Errorf("%w: age must be between 18 and 120", ErrValidation)
	}
	if a.State == "" {
		return fmt.Errorf("%w: state is required", ErrValidation)
	}
	return nil
}

func (in ApplicationInput) Validate() error {
	if in.QuoteID == "" {
		return fmt.Errorf("%w: quote_id is required", ErrValidation)
	}
	return in.Applicant.Validate()
}

// CanTransitionTo checks if a status transition is valid.
func (s ApplicationStatus) CanTransitionTo(next ApplicationStatus) bool {
	transitions := map[ApplicationStatus][]ApplicationStatus{
		ApplicationStatusDraft:       {ApplicationStatusSubmitted},
		ApplicationStatusSubmitted:   {ApplicationStatusUnderReview},
		ApplicationStatusUnderReview: {ApplicationStatusApproved, ApplicationStatusDeclined},
	}
	for _, allowed := range transitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

var (
	ErrApplicationNotFound = fmt.Errorf("%w: application not found", ErrNotFound)
	ErrQuoteAlreadyUsed    = fmt.Errorf("%w: quote already used for an application", ErrConflict)
)
