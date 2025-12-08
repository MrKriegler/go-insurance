package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MrKriegler/go-insurance/internal/platform/ids"
)

type UnderwritingService interface {
	// ProcessApplication is called by the worker when an application is submitted
	ProcessApplication(ctx context.Context, appID string) (UnderwritingCase, error)

	// MakeDecision is called by admin for manual decisions on referred cases
	MakeDecision(ctx context.Context, caseID string, input UWDecisionInput) (UnderwritingCase, error)

	// GetCase retrieves a case by ID
	GetCase(ctx context.Context, caseID string) (UnderwritingCase, error)

	// GetByApplicationID retrieves a case by application ID
	GetByApplicationID(ctx context.Context, appID string) (UnderwritingCase, error)

	// ListReferred returns cases awaiting manual review
	ListReferred(ctx context.Context, limit int) ([]UnderwritingCase, error)
}

type underwritingService struct {
	uw     UnderwritingRepo
	apps   ApplicationRepo
	offers OfferRepo
	clock  func() time.Time
}

func NewUnderwritingService(uw UnderwritingRepo, apps ApplicationRepo, offers OfferRepo) UnderwritingService {
	return &underwritingService{
		uw:     uw,
		apps:   apps,
		offers: offers,
		clock:  time.Now,
	}
}

func (s *underwritingService) ProcessApplication(ctx context.Context, appID string) (UnderwritingCase, error) {
	// 1) Load application
	app, err := s.apps.Get(ctx, appID)
	if err != nil {
		return UnderwritingCase{}, err
	}

	// 2) Verify application is in submitted status
	if app.Status != ApplicationStatusSubmitted {
		return UnderwritingCase{}, fmt.Errorf("%w: application must be in submitted status", ErrInvalidState)
	}

	// 3) Check if UW case already exists
	existing, err := s.uw.GetByApplicationID(ctx, appID)
	if err == nil {
		// Case already exists, return it
		return existing, nil
	}
	if !errors.Is(err, ErrUWCaseNotFound) {
		return UnderwritingCase{}, err
	}

	// 4) Update application status to under_review
	now := s.clock()
	if err := s.apps.UpdateStatus(ctx, appID, ApplicationStatusUnderReview, now); err != nil {
		return UnderwritingCase{}, err
	}

	// 5) Build risk factors
	factors := RiskFactors{
		Age:            app.Applicant.Age,
		Smoker:         app.Applicant.Smoker,
		CoverageAmount: app.CoverageAmount,
		TermYears:      app.TermYears,
	}

	// 6) Score risk
	score := ScoreRisk(factors)

	// 7) Determine decision
	decision, method := s.determineDecision(factors, score)

	// 8) Create UW case
	uwCase := UnderwritingCase{
		ID:            ids.New(),
		ApplicationID: appID,
		RiskFactors:   factors,
		RiskScore:     score,
		Decision:      decision,
		Method:        method,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Set decision details for auto decisions
	if method == UWMethodAuto && decision != UWDecisionReferred {
		uwCase.DecidedBy = "system"
		uwCase.DecidedAt = &now
		if decision == UWDecisionApproved {
			uwCase.Reason = "Auto-approved: meets low-risk criteria"
		} else {
			uwCase.Reason = "Auto-declined: does not meet eligibility requirements"
		}
	}

	// 9) Save UW case
	if err := s.uw.Create(ctx, uwCase); err != nil {
		return UnderwritingCase{}, err
	}

	// 10) If auto-approved, update application and create offer
	if decision == UWDecisionApproved {
		if err := s.apps.UpdateStatus(ctx, appID, ApplicationStatusApproved, now); err != nil {
			return UnderwritingCase{}, err
		}
		if err := s.createOffer(ctx, app, now); err != nil {
			return UnderwritingCase{}, err
		}
	} else if decision == UWDecisionDeclined {
		if err := s.apps.UpdateStatus(ctx, appID, ApplicationStatusDeclined, now); err != nil {
			return UnderwritingCase{}, err
		}
	}
	// If referred, application stays in under_review

	return uwCase, nil
}

func (s *underwritingService) MakeDecision(ctx context.Context, caseID string, input UWDecisionInput) (UnderwritingCase, error) {
	// 1) Validate input
	if err := input.Validate(); err != nil {
		return UnderwritingCase{}, err
	}

	// 2) Load case
	uwCase, err := s.uw.Get(ctx, caseID)
	if err != nil {
		return UnderwritingCase{}, err
	}

	// 3) Verify case can be decided
	if !uwCase.Decision.CanTransitionTo(input.Decision) {
		return UnderwritingCase{}, fmt.Errorf("%w: cannot transition from %s to %s",
			ErrInvalidState, uwCase.Decision, input.Decision)
	}

	// 4) Load application for offer creation if approved
	app, err := s.apps.Get(ctx, uwCase.ApplicationID)
	if err != nil {
		return UnderwritingCase{}, err
	}

	// 5) Update case
	now := s.clock()
	uwCase.Decision = input.Decision
	uwCase.Method = UWMethodManual
	uwCase.DecidedBy = "admin" // In a real system, this would be the admin user ID
	uwCase.Reason = input.Reason
	uwCase.UpdatedAt = now
	uwCase.DecidedAt = &now

	if err := s.uw.Update(ctx, uwCase); err != nil {
		return UnderwritingCase{}, err
	}

	// 6) Update application status
	var newAppStatus ApplicationStatus
	if input.Decision == UWDecisionApproved {
		newAppStatus = ApplicationStatusApproved
	} else {
		newAppStatus = ApplicationStatusDeclined
	}

	if err := s.apps.UpdateStatus(ctx, uwCase.ApplicationID, newAppStatus, now); err != nil {
		return UnderwritingCase{}, err
	}

	// 7) Create offer if approved
	if input.Decision == UWDecisionApproved {
		if err := s.createOffer(ctx, app, now); err != nil {
			return UnderwritingCase{}, err
		}
	}

	return uwCase, nil
}

func (s *underwritingService) GetCase(ctx context.Context, caseID string) (UnderwritingCase, error) {
	if caseID == "" {
		return UnderwritingCase{}, fmt.Errorf("%w: missing case ID", ErrValidation)
	}
	return s.uw.Get(ctx, caseID)
}

func (s *underwritingService) GetByApplicationID(ctx context.Context, appID string) (UnderwritingCase, error) {
	if appID == "" {
		return UnderwritingCase{}, fmt.Errorf("%w: missing application ID", ErrValidation)
	}
	return s.uw.GetByApplicationID(ctx, appID)
}

func (s *underwritingService) ListReferred(ctx context.Context, limit int) ([]UnderwritingCase, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.uw.FindReferred(ctx, limit)
}

func (s *underwritingService) determineDecision(factors RiskFactors, score RiskScore) (UWDecision, UWMethod) {
	// Hard rules - auto-decline
	if ShouldAutoDecline(factors) {
		return UWDecisionDeclined, UWMethodAuto
	}

	// Auto-approve criteria
	if CanAutoApprove(factors, score) {
		return UWDecisionApproved, UWMethodAuto
	}

	// Low-risk auto-approve
	if score.Score <= 20 {
		return UWDecisionApproved, UWMethodAuto
	}

	// Everything else needs manual review
	return UWDecisionReferred, UWMethodAuto
}

func (s *underwritingService) createOffer(ctx context.Context, app Application, now time.Time) error {
	offer := Offer{
		ID:             ids.New(),
		ApplicationID:  app.ID,
		ProductSlug:    app.ProductSlug,
		CoverageAmount: app.CoverageAmount,
		TermYears:      app.TermYears,
		MonthlyPremium: app.MonthlyPremium,
		Status:         OfferStatusPending,
		CreatedAt:      now,
		ExpiresAt:      now.AddDate(0, 0, OfferValidityDays),
	}

	return s.offers.Create(ctx, offer)
}
