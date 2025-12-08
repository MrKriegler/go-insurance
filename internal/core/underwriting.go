package core

import (
	"context"
	"fmt"
	"time"
)

type UWDecision string
type UWMethod string

const (
	UWDecisionPending  UWDecision = "pending"
	UWDecisionApproved UWDecision = "approved"
	UWDecisionDeclined UWDecision = "declined"
	UWDecisionReferred UWDecision = "referred" // Needs manual review
)

const (
	UWMethodAuto   UWMethod = "auto"
	UWMethodManual UWMethod = "manual"
)

// RiskFactors captures scoring inputs.
type RiskFactors struct {
	Age            int   `json:"age"`
	Smoker         bool  `json:"smoker"`
	CoverageAmount int64 `json:"coverage_amount"`
	TermYears      int   `json:"term_years"`
}

// RiskScore is the output of the rules engine.
type RiskScore struct {
	Score       int        `json:"score"`       // 0-100, higher = riskier
	Flags       []string   `json:"flags"`       // e.g., ["high_coverage", "smoker"]
	Recommended UWDecision `json:"recommended"` // Suggested decision
}

// UnderwritingCase tracks the UW process for an application.
type UnderwritingCase struct {
	ID            string      `json:"id"`
	ApplicationID string      `json:"application_id"`
	RiskFactors   RiskFactors `json:"risk_factors"`
	RiskScore     RiskScore   `json:"risk_score"`
	Decision      UWDecision  `json:"decision"`
	Method        UWMethod    `json:"method"` // auto or manual
	DecidedBy     string      `json:"decided_by"` // "system" or admin user ID
	Reason        string      `json:"reason"` // Explanation for decision
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	DecidedAt     *time.Time  `json:"decided_at,omitempty"`
}

type UWDecisionInput struct {
	Decision UWDecision `json:"decision"` // approved or declined
	Reason   string     `json:"reason"`
}

type UnderwritingRepo interface {
	Create(ctx context.Context, uw UnderwritingCase) error
	Get(ctx context.Context, id string) (UnderwritingCase, error)
	GetByApplicationID(ctx context.Context, appID string) (UnderwritingCase, error)
	Update(ctx context.Context, uw UnderwritingCase) error
	FindPending(ctx context.Context, limit int) ([]UnderwritingCase, error)
	FindReferred(ctx context.Context, limit int) ([]UnderwritingCase, error)
}

func (in UWDecisionInput) Validate() error {
	if in.Decision != UWDecisionApproved && in.Decision != UWDecisionDeclined {
		return fmt.Errorf("%w: decision must be 'approved' or 'declined'", ErrValidation)
	}
	if in.Reason == "" {
		return fmt.Errorf("%w: reason is required", ErrValidation)
	}
	return nil
}

// CanTransitionTo checks if a decision transition is valid.
func (d UWDecision) CanTransitionTo(next UWDecision) bool {
	transitions := map[UWDecision][]UWDecision{
		UWDecisionPending:  {UWDecisionApproved, UWDecisionDeclined, UWDecisionReferred},
		UWDecisionReferred: {UWDecisionApproved, UWDecisionDeclined},
	}
	for _, allowed := range transitions[d] {
		if allowed == next {
			return true
		}
	}
	return false
}

// ScoreRisk calculates the risk score based on factors.
func ScoreRisk(factors RiskFactors) RiskScore {
	score := 0
	var flags []string

	// Age scoring
	switch {
	case factors.Age > 80:
		// Hard decline - handled separately in decision logic
		return RiskScore{Score: 100, Flags: []string{"age_over_80"}, Recommended: UWDecisionDeclined}
	case factors.Age > 65:
		score += 50
		flags = append(flags, "senior_65_plus")
	case factors.Age > 60:
		score += 35
		flags = append(flags, "senior")
	case factors.Age > 50:
		score += 25
	case factors.Age > 40:
		score += 10
	}

	// Smoker scoring
	if factors.Smoker {
		score += 25
		flags = append(flags, "smoker")
	}

	// Coverage amount scoring
	switch {
	case factors.CoverageAmount > 500000:
		score += 25
		flags = append(flags, "high_coverage")
	case factors.CoverageAmount > 250000:
		score += 15
		flags = append(flags, "medium_high_coverage")
	case factors.CoverageAmount > 100000:
		score += 10
	}

	// Determine recommendation
	var recommended UWDecision
	switch {
	case score <= 20:
		recommended = UWDecisionApproved
	case score <= 50:
		recommended = UWDecisionReferred
	default:
		recommended = UWDecisionReferred
	}

	return RiskScore{Score: score, Flags: flags, Recommended: recommended}
}

// CanAutoApprove checks if factors qualify for automatic approval.
func CanAutoApprove(factors RiskFactors, score RiskScore) bool {
	return factors.Age < 45 &&
		!factors.Smoker &&
		factors.CoverageAmount < 250000 &&
		score.Score <= 30
}

// ShouldAutoDecline checks if factors require automatic decline.
func ShouldAutoDecline(factors RiskFactors) bool {
	return factors.Age > 80
}

var (
	ErrUWCaseNotFound    = fmt.Errorf("%w: underwriting case not found", ErrNotFound)
	ErrUWCaseExists      = fmt.Errorf("%w: underwriting case already exists for application", ErrConflict)
	ErrUWAlreadyDecided  = fmt.Errorf("%w: underwriting case already decided", ErrInvalidState)
	ErrUWInvalidDecision = fmt.Errorf("%w: invalid underwriting decision", ErrValidation)
)
