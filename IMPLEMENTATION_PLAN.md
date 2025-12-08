# Go Insurance API - Portfolio Demo Implementation Plan

## Overview
Complete the insurance API with full workflow: **Quote → Application → Underwriting → Offer → Policy**

Features:
- Auto-approve simple cases, refer complex ones for manual review
- Background workers for async processing
- Clean architecture with domain services

---

## Implementation Phases

### Phase 1: Domain Models (`internal/core/`)

**1.1 `applications.go`** - Application entity
- Status: `draft` → `submitted` → `under_review` → `approved`/`declined`
- Fields: ID, QuoteID, Applicant (name, email, DOB, age, smoker, state), status, timestamps
- Validation: email format, age 18-120, required fields
- Repo interface: Create, Get, Update, UpdateStatus, FindByStatus

**1.2 `underwriting.go`** - Underwriting case entity
- Decision: `pending` → `approved`/`declined`/`referred`
- Method: `auto` or `manual`
- RiskFactors: age, smoker, coverage, term
- RiskScore: score (0-100), flags, recommended decision
- Repo interface: Create, Get, GetByApplicationID, Update, FindPending, FindReferred

**1.3 `offers.go`** - Offer entity
- Status: `pending` → `accepted`/`declined`/`expired`
- 30-day expiry from creation
- Repo interface: Create, Get, GetByApplicationID, Update, FindAccepted

**1.4 `policies.go`** - Policy entity
- Status: `active` (future: lapsed, cancelled, expired)
- Human-readable number: `POL-2024-000001`
- EffectiveDate, ExpiryDate (effective + term years)
- Repo interface: Create, Get, GetByNumber, List, NextPolicyNumber

---

### Phase 2: MongoDB Repositories (`internal/store/mongo/`)

**2.1 `types.go`** - Add document structs
- ApplicantDoc, ApplicationDoc, UnderwritingCaseDoc, OfferDoc, PolicyDoc
- Conversion functions: toXxxDoc / fromXxxDoc

**2.2 `applications_repo.go`**
- Collection: `applications`
- Indexes: quote_id (unique), status

**2.3 `underwriting_repo.go`**
- Collection: `underwriting_cases`
- Indexes: application_id (unique), decision

**2.4 `offers_repo.go`**
- Collection: `offers`
- Indexes: application_id (unique), status, expires_at (TTL optional)

**2.5 `policies_repo.go`**
- Collection: `policies`
- Indexes: number (unique), application_id

**2.6 `indexes.go`** - Add new indexes

---

### Phase 3: Services (`internal/core/`)

**3.1 `application_service.go`**
- Create: validate quote exists & not expired, create draft
- Patch: only in draft status
- Submit: validate complete, transition to submitted

**3.2 `underwriting_service.go`**
- ProcessApplication: called by worker
  1. Create UW case with risk factors
  2. Score risk
  3. Apply decision rules
  4. Update application status
  5. If approved → auto-generate offer
- MakeDecision: admin manual decision
- Risk scoring rules:
  - Age > 80 → auto-decline
  - Age < 45, non-smoker, coverage < $250k, score ≤ 30 → auto-approve
  - Score ≤ 20 → auto-approve
  - Everything else → referred for manual review

**3.3 `offer_service.go`**
- GenerateOffer: create from approved application
- Accept: validate not expired, transition status
- Decline: transition status

**3.4 `policy_service.go`**
- IssueFromOffer: called by worker
  1. Load accepted offer
  2. Generate policy number
  3. Set effective/expiry dates
  4. Create policy record
  5. Mark offer as processed (or leave as accepted)

---

### Phase 4: HTTP Handlers (`internal/http/handlers/`)

**4.1 `applications.go`**
```
POST   /applications              → Create draft application
GET    /applications/{id}         → Get application
PATCH  /applications/{id}         → Update applicant info (draft only)
POST   /applications/{id}:submit  → Submit for underwriting
```

**4.2 `uw.go`**
```
GET    /underwriting/cases/{id}      → Get UW case details
GET    /underwriting/cases           → List referred cases (admin)
POST   /underwriting/cases/{id}:decide → Manual decision (admin)
```

**4.3 `offers.go`**
```
POST   /applications/{id}/offers  → Generate offer (after approval)
GET    /offers/{id}               → Get offer
POST   /offers/{id}:accept        → Accept offer
POST   /offers/{id}:decline       → Decline offer
```

**4.4 `policies.go`**
```
GET    /policies/{number}  → Get policy by number
GET    /policies           → List policies (with filters)
```

---

### Phase 5: Background Workers (`internal/jobs/`)

**5.1 `worker.go`** - Base worker with polling loop
```go
type Worker interface {
    Start(ctx context.Context)
    Name() string
}
```
- Ticker-based polling with configurable interval
- Graceful shutdown via context cancellation
- Structured logging

**5.2 `underwriting_worker.go`**
- Poll interval: 5 seconds
- Find applications with status `submitted`
- Call UnderwritingService.ProcessApplication for each

**5.3 `issuance_worker.go`**
- Poll interval: 5 seconds
- Find offers with status `accepted`
- Call PolicyService.IssueFromOffer for each

---

### Phase 6: Integration (`cmd/api/main.go`)

1. Initialize all repositories
2. Initialize all services with dependencies
3. Initialize handlers with services
4. Mount handlers on router
5. Start workers as goroutines
6. Add worker config (poll intervals) to config.go

---

## Auto-Underwriting Decision Matrix

| Criteria | Action |
|----------|--------|
| Age > 80 | Auto-decline |
| Age < 45 AND non-smoker AND coverage < $250k AND score ≤ 30 | Auto-approve |
| Score ≤ 20 | Auto-approve |
| All other cases | Refer to manual review |

### Risk Score Components
- Age 51-60: +25 | Age 61-65: +35 | Age 66-80: +50
- Smoker: +25
- Coverage $100k-$250k: +10 | $250k-$500k: +15 | > $500k: +25

---

## E2E Workflow Test

```bash
# 1. Create quote
POST /api/v1/quotes
{"product_slug":"term-life-10","coverage_amount":150000,"term_years":10,"age":35,"smoker":false}

# 2. Create application
POST /api/v1/applications
{"quote_id":"<quote_id>","applicant":{"first_name":"John","last_name":"Doe","email":"john@example.com","date_of_birth":"1989-06-15","age":35,"smoker":false,"state":"CA"}}

# 3. Submit application
POST /api/v1/applications/<app_id>:submit

# 4. [Worker auto-approves, creates offer]

# 5. Accept offer
POST /api/v1/offers/<offer_id>:accept

# 6. [Worker issues policy]

# 7. Get policy
GET /api/v1/policies/POL-2024-000001
```
