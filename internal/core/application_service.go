package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/platform/ids"
)

type ApplicationService interface {
	Create(ctx context.Context, in ApplicationInput) (Application, error)
	Get(ctx context.Context, id string) (Application, error)
	Patch(ctx context.Context, id string, patch ApplicationPatch) (Application, error)
	Submit(ctx context.Context, id string) (Application, error)
}

type applicationService struct {
	apps   ApplicationRepo
	quotes QuoteRepo
	clock  func() time.Time
}

func NewApplicationService(apps ApplicationRepo, quotes QuoteRepo) ApplicationService {
	return &applicationService{
		apps:   apps,
		quotes: quotes,
		clock:  time.Now,
	}
}

func (s *applicationService) Create(ctx context.Context, in ApplicationInput) (Application, error) {
	// 1) Validate input
	if err := in.Validate(); err != nil {
		return Application{}, err
	}

	// 2) Load quote
	quote, err := s.quotes.Get(ctx, in.QuoteID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Application{}, fmt.Errorf("%w: quote %q", ErrNotFound, in.QuoteID)
		}
		return Application{}, err
	}

	// 3) Check quote is not expired
	now := s.clock()
	if now.After(quote.ExpiresAt) {
		return Application{}, fmt.Errorf("%w: quote has expired", ErrInvalidState)
	}

	// 4) Create application
	app := Application{
		ID:             ids.New(),
		QuoteID:        quote.ID,
		ProductID:      quote.ProductID,
		ProductSlug:    quote.ProductSlug,
		CoverageAmount: quote.CoverageAmount,
		TermYears:      quote.TermYears,
		MonthlyPremium: quote.MonthlyPremium,
		Applicant:      in.Applicant,
		Status:         ApplicationStatusDraft,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// 5) Persist
	if err := s.apps.Create(ctx, app); err != nil {
		if errors.Is(err, ErrConflict) {
			return Application{}, ErrQuoteAlreadyUsed
		}
		return Application{}, err
	}

	return app, nil
}

func (s *applicationService) Get(ctx context.Context, id string) (Application, error) {
	if id == "" {
		return Application{}, fmt.Errorf("%w: missing application ID", ErrValidation)
	}
	return s.apps.Get(ctx, id)
}

func (s *applicationService) Patch(ctx context.Context, id string, patch ApplicationPatch) (Application, error) {
	// 1) Load application
	app, err := s.apps.Get(ctx, id)
	if err != nil {
		return Application{}, err
	}

	// 2) Only allow patching in draft status
	if app.Status != ApplicationStatusDraft {
		return Application{}, fmt.Errorf("%w: can only update applications in draft status", ErrInvalidState)
	}

	// 3) Apply patch
	if patch.Applicant != nil {
		if err := patch.Applicant.Validate(); err != nil {
			return Application{}, err
		}
		app.Applicant = *patch.Applicant
	}

	app.UpdatedAt = s.clock()

	// 4) Persist
	if err := s.apps.Update(ctx, app); err != nil {
		return Application{}, err
	}

	return app, nil
}

func (s *applicationService) Submit(ctx context.Context, id string) (Application, error) {
	// 1) Load application
	app, err := s.apps.Get(ctx, id)
	if err != nil {
		return Application{}, err
	}

	// 2) Validate current status allows submission
	if !app.Status.CanTransitionTo(ApplicationStatusSubmitted) {
		return Application{}, fmt.Errorf("%w: cannot submit application in %s status", ErrInvalidState, app.Status)
	}

	// 3) Validate application is complete
	if err := app.Applicant.Validate(); err != nil {
		return Application{}, fmt.Errorf("%w: application incomplete - %v", ErrValidation, err)
	}

	// 4) Update status
	now := s.clock()
	app.Status = ApplicationStatusSubmitted
	app.UpdatedAt = now
	app.SubmittedAt = &now

	// 5) Persist
	if err := s.apps.Update(ctx, app); err != nil {
		return Application{}, err
	}

	return app, nil
}
