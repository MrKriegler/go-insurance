# Go Insurance API - Frontend Tech Demo Design

A comprehensive blueprint for building a Next.js/React frontend that demonstrates the Life Insurance Quote and Policy Management API.

---

## Table of Contents

1. [Site Architecture](#1-site-architecture)
2. [Landing Page](#2-landing-page)
3. [Guided Journey](#3-guided-journey)
4. [API Playground](#4-api-playground)
5. [Underwriter Workbench](#5-underwriter-workbench)
6. [Policy Dashboard](#6-policy-dashboard)
7. [Developer Docs](#7-developer-docs)
8. [Technical Overview](#8-technical-overview)
9. [Component Structure](#9-component-structure)
10. [API Integration Layer](#10-api-integration-layer)

---

## 1. Site Architecture

### Route Structure

```
/                           Landing page
/demo/journey               Guided end-to-end journey
/demo/playground            API playground (Postman-style)
/demo/underwriting          Underwriter workbench
/demo/policies              Policy & offer dashboard
/docs                       Developer documentation
/docs/products              Products API docs
/docs/quotes                Quotes API docs
/docs/applications          Applications API docs
/docs/underwriting          Underwriting API docs
/docs/offers                Offers API docs
/docs/policies              Policies API docs
/tech                       Technical overview
```

### Navigation Structure

```
Logo | Home | Demo (dropdown) | Docs | Tech | GitHub
                |
                +-- Journey
                +-- Playground
                +-- Underwriting
                +-- Policies
```

### Tech Stack

- **Framework**: Next.js 14 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Components**: shadcn/ui
- **State**: React Query (TanStack Query)
- **Code Editor**: Monaco Editor (for JSON editing)
- **HTTP Client**: Native fetch with typed wrappers

---

## 2. Landing Page

**Route**: `/`

**Purpose**: Introduce the API, explain the insurance lifecycle, and invite users to explore.

**Primary Audience**: Product owners, business stakeholders, developers evaluating the API.

### Layout

```
+------------------------------------------------------------------+
|  Navigation Bar                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  [Hero Section]                                                   |
|  Headline + Subheadline + CTA buttons                            |
|                                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  [Lifecycle Visualization]                                        |
|  Products -> Quotes -> Applications -> UW -> Offers -> Policies  |
|                                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  [Feature Cards - 3 column grid]                                 |
|  Auto-Underwriting | Background Workers | REST API               |
|                                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  [Code Preview Section]                                          |
|  Sample curl command with response                               |
|                                                                   |
+------------------------------------------------------------------+
|  Footer                                                          |
+------------------------------------------------------------------+
```

### Copy

**Hero Section**:
```
Headline:       Life Insurance API
Subheadline:    A complete quote-to-policy management system.
                Built with Go. Ready for production.

CTA Primary:    Start Demo
CTA Secondary:  View Documentation
```

**Lifecycle Section**:
```
Headline:       From Quote to Policy in Six Steps

Step 1: Products
Browse available term life and whole life products with coverage limits and base rates.

Step 2: Quotes
Get instant pricing based on age, coverage amount, term, and smoking status.

Step 3: Applications
Create an application linking a quote to applicant personal information.

Step 4: Underwriting
Automatic risk scoring with auto-approve, auto-decline, or referral for manual review.

Step 5: Offers
Approved applications receive binding offers valid for 30 days.

Step 6: Policies
Accepted offers become active policies with unique policy numbers.
```

**Feature Cards**:
```
Card 1: Automatic Underwriting
Risk scoring engine that auto-approves low-risk applicants and refers complex cases for manual review.

Card 2: Background Processing
Async workers handle underwriting decisions and policy issuance without blocking API responses.

Card 3: RESTful Design
Clean JSON API with RFC 7807 error responses, OpenAPI documentation, and predictable resource paths.
```

---

## 3. Guided Journey

**Route**: `/demo/journey`

**Purpose**: Walk users through the complete insurance lifecycle with a real API.

**Primary Audience**: Product owners, business stakeholders, developers learning the flow.

### Layout

```
+------------------------------------------------------------------+
|  Navigation Bar                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  [Progress Stepper]                                              |
|  (1) Products > (2) Quote > (3) Application > (4) Submit >       |
|  (5) Underwriting > (6) Offer > (7) Policy                       |
|                                                                   |
+------------------------------------------------------------------+
|                          |                                        |
|  [Left Panel - 50%]      |  [Right Panel - 50%]                  |
|  Interactive Form        |  API Inspector                        |
|                          |                                        |
|  - Form fields           |  - Endpoint: POST /quotes             |
|  - Validation            |  - Request JSON                       |
|  - Action button         |  - Response JSON                      |
|                          |  - Status + Latency                   |
|                          |                                        |
+------------------------------------------------------------------+
|  [Bottom Bar]                                                    |
|  Previous Step | Current IDs display | Next Step                 |
+------------------------------------------------------------------+
```

### Step-by-Step Flow

#### Step 1: Browse Products

**UI Panel**:
- Product cards in a grid
- Each card shows: name, term, coverage range, base rate
- "Select Product" button on each card

**API Panel**:
```
Endpoint: GET /api/v1/products
Request: (none)
Response:
[
  {
    "id": "01JEGXYZ...",
    "slug": "term-life-10",
    "name": "10-Year Term Life",
    "term_years": 10,
    "min_coverage": 50000,
    "max_coverage": 500000,
    "base_rate": 0.25
  },
  ...
]
```

#### Step 2: Get a Quote

**UI Panel**:
- Selected product displayed
- Form fields:
  - Coverage Amount (slider: $50,000 - $500,000)
  - Age (number input: 18-80)
  - Smoker (toggle: Yes/No)
- "Get Quote" button
- Result card showing monthly premium

**API Panel**:
```
Endpoint: POST /api/v1/quotes
Request:
{
  "product_slug": "term-life-10",
  "coverage_amount": 150000,
  "term_years": 10,
  "age": 35,
  "smoker": false
}

Response:
{
  "id": "01JEH1ABC...",
  "product_id": "01JEGXYZ...",
  "product_slug": "term-life-10",
  "coverage_amount": 150000,
  "term_years": 10,
  "monthly_premium": 37.50,
  "status": "priced",
  "created_at": "2025-12-06T23:15:00Z",
  "expires_at": "2025-12-07T23:15:00Z"
}
```

**ID Tracking**: `quote_id: 01JEH1ABC...`

#### Step 3: Create Application

**UI Panel**:
- Quote summary card
- Applicant form:
  - First Name, Last Name
  - Email
  - Date of Birth
  - Age (auto-calculated or manual)
  - Smoker (toggle)
  - State (dropdown: US states)
- "Create Application" button

**API Panel**:
```
Endpoint: POST /api/v1/applications
Request:
{
  "quote_id": "01JEH1ABC...",
  "applicant": {
    "first_name": "John",
    "last_name": "Doe",
    "email": "john.doe@example.com",
    "date_of_birth": "1989-06-15",
    "age": 35,
    "smoker": false,
    "state": "CA"
  }
}

Response:
{
  "id": "01JEH2DEF...",
  "quote_id": "01JEH1ABC...",
  "product_slug": "term-life-10",
  "coverage_amount": 150000,
  "term_years": 10,
  "monthly_premium": 37.50,
  "applicant": {
    "first_name": "John",
    "last_name": "Doe",
    "email": "john.doe@example.com",
    "date_of_birth": "1989-06-15",
    "age": 35,
    "smoker": false,
    "state": "CA"
  },
  "status": "draft",
  "created_at": "2025-12-06T23:16:00Z",
  "updated_at": "2025-12-06T23:16:00Z"
}
```

**ID Tracking**: `quote_id: 01JEH1ABC... → application_id: 01JEH2DEF...`

#### Step 4: Submit for Underwriting

**UI Panel**:
- Application summary
- Checklist showing completeness
- "Submit Application" button
- Status indicator: Draft → Submitted

**API Panel**:
```
Endpoint: POST /api/v1/applications/01JEH2DEF...:submit
Request: (none - action endpoint)

Response:
{
  "id": "01JEH2DEF...",
  "quote_id": "01JEH1ABC...",
  ...
  "status": "submitted",
  "submitted_at": "2025-12-06T23:17:00Z"
}
```

#### Step 5: Underwriting Decision

**UI Panel**:
- Animated "Processing..." state
- After worker processes (poll every 2 seconds):
  - Risk score visualization
  - Decision badge: APPROVED / DECLINED / REFERRED
  - Risk factors breakdown
  - If referred: link to Underwriter Workbench

**API Panel** (polling):
```
Endpoint: GET /api/v1/applications/01JEH2DEF...
Response shows status change: submitted → under_review → approved

Then fetch UW case:
Endpoint: GET /api/v1/underwriting/cases (filter by application)

Response:
{
  "id": "01JEH3GHI...",
  "application_id": "01JEH2DEF...",
  "risk_factors": {
    "age": 35,
    "smoker": false,
    "coverage_amount": 150000,
    "term_years": 10
  },
  "risk_score": {
    "score": 10,
    "flags": [],
    "recommended": "approved"
  },
  "decision": "approved",
  "method": "auto",
  "decided_by": "system",
  "reason": "Low risk profile - auto approved",
  "created_at": "2025-12-06T23:17:05Z",
  "decided_at": "2025-12-06T23:17:05Z"
}
```

**ID Tracking**: `application_id: 01JEH2DEF... → uw_case_id: 01JEH3GHI...`

#### Step 6: Generate & Accept Offer

**UI Panel**:
- Offer card showing:
  - Coverage amount
  - Monthly premium
  - Expiration date (30 days)
  - Status badge
- "Accept Offer" / "Decline Offer" buttons

**API Panel**:
```
Generate Offer:
Endpoint: POST /api/v1/applications/01JEH2DEF.../offers

Response:
{
  "id": "01JEH4JKL...",
  "application_id": "01JEH2DEF...",
  "product_slug": "term-life-10",
  "coverage_amount": 150000,
  "term_years": 10,
  "monthly_premium": 37.50,
  "status": "pending",
  "created_at": "2025-12-06T23:17:10Z",
  "expires_at": "2025-01-05T23:17:10Z"
}

Accept Offer:
Endpoint: POST /api/v1/offers/01JEH4JKL...:accept

Response:
{
  "id": "01JEH4JKL...",
  ...
  "status": "accepted",
  "accepted_at": "2025-12-06T23:18:00Z"
}
```

**ID Tracking**: `application_id: 01JEH2DEF... → offer_id: 01JEH4JKL...`

#### Step 7: View Issued Policy

**UI Panel**:
- Policy certificate visualization
- Policy number prominently displayed
- Coverage details
- Effective and expiry dates
- Insured information
- "Download" button (mock)

**API Panel** (poll for issuance):
```
Endpoint: GET /api/v1/policies

Response:
{
  "items": [
    {
      "id": "01JEH5MNO...",
      "number": "POL-2025-000001",
      "application_id": "01JEH2DEF...",
      "offer_id": "01JEH4JKL...",
      "product_slug": "term-life-10",
      "coverage_amount": 150000,
      "term_years": 10,
      "monthly_premium": 37.50,
      "insured": {
        "first_name": "John",
        "last_name": "Doe",
        "email": "john.doe@example.com",
        "date_of_birth": "1989-06-15",
        "age": 35,
        "smoker": false,
        "state": "CA"
      },
      "status": "active",
      "effective_date": "2025-12-07T00:00:00Z",
      "expiry_date": "2035-12-07T00:00:00Z",
      "issued_at": "2025-12-06T23:18:05Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

**Final ID Chain**:
```
quote_id:       01JEH1ABC...
application_id: 01JEH2DEF...
uw_case_id:     01JEH3GHI...
offer_id:       01JEH4JKL...
policy_id:      01JEH5MNO...
policy_number:  POL-2025-000001
```

### Bottom Bar - ID Tracker

Always visible component showing the chain of IDs created during the journey:

```
+------------------------------------------------------------------+
| Quote: 01JEH1ABC | App: 01JEH2DEF | Offer: 01JEH4JKL | Policy: POL-2025-000001 |
+------------------------------------------------------------------+
```

Each ID is clickable to view that resource's current state.

---

## 4. API Playground

**Route**: `/demo/playground`

**Purpose**: Postman-like interface for exploring all API endpoints.

**Primary Audience**: Developers integrating with the API.

### Layout

```
+------------------------------------------------------------------+
|  Navigation Bar                                                   |
+------------------------------------------------------------------+
|          |                              |                         |
| [Left]   |  [Center - 50%]              |  [Right - 30%]         |
| Sidebar  |  Request Builder             |  Response Viewer       |
| 20%      |                              |                         |
|          |  +------------------------+  |  +-------------------+ |
| Scenarios|  | GET  | /api/v1/products|  |  | 200 OK    45ms   | |
|          |  +------------------------+  |  +-------------------+ |
| Products |  |                        |  |  |                   | |
|  - List  |  | Path Parameters:       |  |  | [                 | |
|  - Get   |  | (none)                 |  |  |   {               | |
|          |  |                        |  |  |     "id": "...",  | |
| Quotes   |  | Query Parameters:      |  |  |     "slug": "..." | |
|  - Create|  | (none)                 |  |  |   }               | |
|  - Get   |  |                        |  |  | ]                 | |
|          |  | Request Body:          |  |  |                   | |
| Apps     |  | (none for GET)         |  |  +-------------------+ |
|  - Create|  |                        |  |                         |
|  - Get   |  | [    Send Request    ] |  |  Summary:              |
|  - Patch |  +------------------------+  |  5 products returned   |
|  - Submit|                              |                         |
|          |                              |                         |
| UW Cases |                              |                         |
|  - List  |                              |                         |
|  - Get   |                              |                         |
|  - Decide|                              |                         |
|          |                              |                         |
| Offers   |                              |                         |
|  - Create|                              |                         |
|  - Get   |                              |                         |
|  - Accept|                              |                         |
|  - Decline                              |                         |
|          |                              |                         |
| Policies |                              |                         |
|  - List  |                              |                         |
|  - Get   |                              |                         |
+------------------------------------------------------------------+
```

### Scenario Definitions

```typescript
const scenarios = [
  // Products
  {
    group: "Products",
    name: "List all products",
    method: "GET",
    path: "/products",
    pathParams: [],
    queryParams: [],
    body: null,
  },
  {
    group: "Products",
    name: "Get product by slug",
    method: "GET",
    path: "/products/{product_slug}",
    pathParams: [{ name: "product_slug", default: "term-life-10" }],
    queryParams: [],
    body: null,
  },

  // Quotes
  {
    group: "Quotes",
    name: "Create quote",
    method: "POST",
    path: "/quotes",
    pathParams: [],
    queryParams: [],
    body: {
      product_slug: "term-life-10",
      coverage_amount: 150000,
      term_years: 10,
      age: 35,
      smoker: false
    },
  },
  {
    group: "Quotes",
    name: "Get quote",
    method: "GET",
    path: "/quotes/{quote_id}",
    pathParams: [{ name: "quote_id", default: "" }],
    queryParams: [],
    body: null,
  },

  // Applications
  {
    group: "Applications",
    name: "Create application",
    method: "POST",
    path: "/applications",
    pathParams: [],
    queryParams: [],
    body: {
      quote_id: "",
      applicant: {
        first_name: "John",
        last_name: "Doe",
        email: "john@example.com",
        date_of_birth: "1989-06-15",
        age: 35,
        smoker: false,
        state: "CA"
      }
    },
  },
  {
    group: "Applications",
    name: "Get application",
    method: "GET",
    path: "/applications/{application_id}",
    pathParams: [{ name: "application_id", default: "" }],
    queryParams: [],
    body: null,
  },
  {
    group: "Applications",
    name: "Update application",
    method: "PATCH",
    path: "/applications/{application_id}",
    pathParams: [{ name: "application_id", default: "" }],
    queryParams: [],
    body: {
      applicant: {
        first_name: "Jane",
        last_name: "Doe",
        email: "jane@example.com",
        date_of_birth: "1989-06-15",
        age: 35,
        smoker: false,
        state: "NY"
      }
    },
  },
  {
    group: "Applications",
    name: "Submit application",
    method: "POST",
    path: "/applications/{application_id}:submit",
    pathParams: [{ name: "application_id", default: "" }],
    queryParams: [],
    body: null,
  },

  // Underwriting
  {
    group: "Underwriting",
    name: "List referred cases",
    method: "GET",
    path: "/underwriting/cases",
    pathParams: [],
    queryParams: [],
    body: null,
  },
  {
    group: "Underwriting",
    name: "Get case details",
    method: "GET",
    path: "/underwriting/cases/{case_id}",
    pathParams: [{ name: "case_id", default: "" }],
    queryParams: [],
    body: null,
  },
  {
    group: "Underwriting",
    name: "Decide case",
    method: "POST",
    path: "/underwriting/cases/{case_id}:decide",
    pathParams: [{ name: "case_id", default: "" }],
    queryParams: [],
    body: {
      decision: "approved",
      reason: "Manual review complete - risk acceptable"
    },
  },

  // Offers
  {
    group: "Offers",
    name: "Generate offer",
    method: "POST",
    path: "/applications/{application_id}/offers",
    pathParams: [{ name: "application_id", default: "" }],
    queryParams: [],
    body: null,
  },
  {
    group: "Offers",
    name: "Get offer",
    method: "GET",
    path: "/offers/{offer_id}",
    pathParams: [{ name: "offer_id", default: "" }],
    queryParams: [],
    body: null,
  },
  {
    group: "Offers",
    name: "Accept offer",
    method: "POST",
    path: "/offers/{offer_id}:accept",
    pathParams: [{ name: "offer_id", default: "" }],
    queryParams: [],
    body: null,
  },
  {
    group: "Offers",
    name: "Decline offer",
    method: "POST",
    path: "/offers/{offer_id}:decline",
    pathParams: [{ name: "offer_id", default: "" }],
    queryParams: [],
    body: null,
  },

  // Policies
  {
    group: "Policies",
    name: "List policies",
    method: "GET",
    path: "/policies",
    pathParams: [],
    queryParams: [
      { name: "status", default: "" },
      { name: "application_id", default: "" },
      { name: "limit", default: "20" },
      { name: "offset", default: "0" },
    ],
    body: null,
  },
  {
    group: "Policies",
    name: "Get policy by number",
    method: "GET",
    path: "/policies/{policy_number}",
    pathParams: [{ name: "policy_number", default: "POL-2025-000001" }],
    queryParams: [],
    body: null,
  },
];
```

### Configuration Panel

At the top of the playground:

```
Base URL: [ http://localhost:8080/api/v1 ]  [Reset to Default]

[ ] Auto-fill IDs from previous responses
```

When "Auto-fill IDs" is enabled, the playground tracks IDs from responses and auto-populates them in subsequent requests.

---

## 5. Underwriter Workbench

**Route**: `/demo/underwriting`

**Purpose**: Simulate an underwriter's daily workflow for reviewing referred cases.

**Primary Audience**: Business stakeholders, underwriters, product owners.

### Layout

```
+------------------------------------------------------------------+
|  Navigation Bar                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  Underwriter Workbench                          [Refresh Queue]  |
|                                                                   |
+------------------------------------------------------------------+
|                          |                                        |
|  [Case Queue - 35%]      |  [Case Details - 65%]                 |
|                          |                                        |
|  Referred Cases (3)      |  Case: 01JEH3GHI...                   |
|  +--------------------+  |                                        |
|  | John Doe           |  |  Application Details                  |
|  | $250,000 coverage  |  |  +--------------------------------+   |
|  | Score: 45          |  |  | Name: John Doe                 |   |
|  | [REFERRED]         |  |  | Email: john@example.com        |   |
|  +--------------------+  |  | Age: 55 | Smoker: No           |   |
|  | Jane Smith         |  |  | Coverage: $250,000             |   |
|  | $500,000 coverage  |  |  | Term: 20 years                 |   |
|  | Score: 65          |  |  | Premium: $87.50/mo             |   |
|  | [REFERRED]         |  |  +--------------------------------+   |
|  +--------------------+  |                                        |
|  | Bob Wilson         |  |  Risk Assessment                      |
|  | $750,000 coverage  |  |  +--------------------------------+   |
|  | Score: 72          |  |  | Score: 45 / 100                |   |
|  | [REFERRED]         |  |  | [==========          ]         |   |
|  +--------------------+  |  |                                |   |
|                          |  | Flags:                         |   |
|                          |  | - senior (age 51-60)           |   |
|                          |  | - medium_high_coverage         |   |
|                          |  |                                |   |
|                          |  | Recommendation: REFER          |   |
|                          |  +--------------------------------+   |
|                          |                                        |
|                          |  Decision                              |
|                          |  +--------------------------------+   |
|                          |  | ( ) Approve  ( ) Decline       |   |
|                          |  |                                |   |
|                          |  | Reason:                        |   |
|                          |  | [                            ] |   |
|                          |  |                                |   |
|                          |  | [    Submit Decision    ]      |   |
|                          |  +--------------------------------+   |
|                          |                                        |
+------------------------------------------------------------------+
```

### API Endpoints Used

**Load Queue**:
```
GET /api/v1/underwriting/cases
```

**Load Case Details**:
```
GET /api/v1/underwriting/cases/{case_id}
```

**Submit Decision**:
```
POST /api/v1/underwriting/cases/{case_id}:decide
Body:
{
  "decision": "approved",  // or "declined"
  "reason": "Underwriter notes here..."
}
```

### Copy

```
Page Title: Underwriter Workbench

Queue Header: Referred Cases
Queue Empty State: No cases pending review. All caught up!

Details Header: Case Details
Decision Section: Make a Decision

Approve Button: Approve Application
Decline Button: Decline Application
Reason Label: Decision Reason (required)
Reason Placeholder: Enter your underwriting notes...

Submit Button: Submit Decision

Success Toast: Decision recorded. Case moved to {approved/declined}.
Error Toast: Failed to submit decision. Please try again.
```

---

## 6. Policy Dashboard

**Route**: `/demo/policies`

**Purpose**: View issued policies and their relationships to applications/offers.

**Primary Audience**: Business stakeholders, operations, customer service.

### Layout

```
+------------------------------------------------------------------+
|  Navigation Bar                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  Policy Dashboard                                                |
|                                                                   |
|  [Filters]                                                       |
|  Status: [All v]  Application ID: [________]  [Search]          |
|                                                                   |
+------------------------------------------------------------------+
|                                                                   |
|  +-------------------------------------------------------------+ |
|  | Policy #      | Insured    | Coverage   | Premium | Status  | |
|  +-------------------------------------------------------------+ |
|  | POL-2025-0001 | John Doe   | $150,000   | $37.50  | Active  | |
|  | POL-2025-0002 | Jane Smith | $250,000   | $62.50  | Active  | |
|  | POL-2025-0003 | Bob Wilson | $100,000   | $25.00  | Active  | |
|  +-------------------------------------------------------------+ |
|                                                                   |
|  Showing 1-3 of 3 policies          [< Prev] [1] [Next >]       |
|                                                                   |
+------------------------------------------------------------------+

[When a row is clicked, slide-out panel appears:]

+------------------------------------------------------------------+
|                                         |                         |
|  [Table - 70%]                          |  [Details Panel - 30%] |
|                                         |                         |
|                                         |  Policy POL-2025-0001  |
|                                         |                         |
|                                         |  Status: Active        |
|                                         |  Issued: Dec 6, 2024   |
|                                         |                         |
|                                         |  Coverage              |
|                                         |  Amount: $150,000      |
|                                         |  Term: 10 years        |
|                                         |  Premium: $37.50/mo    |
|                                         |                         |
|                                         |  Dates                 |
|                                         |  Effective: Dec 7, 2024|
|                                         |  Expires: Dec 7, 2034  |
|                                         |                         |
|                                         |  Insured               |
|                                         |  John Doe              |
|                                         |  john@example.com      |
|                                         |  DOB: Jun 15, 1989     |
|                                         |  State: CA             |
|                                         |                         |
|                                         |  Related IDs           |
|                                         |  App: 01JEH2DEF...     |
|                                         |  Offer: 01JEH4JKL...   |
|                                         |                         |
|                                         |  [View Application]    |
|                                         |  [View in Playground]  |
+------------------------------------------------------------------+
```

### API Endpoints Used

**List Policies**:
```
GET /api/v1/policies?status=active&limit=20&offset=0
```

**Get Policy Details**:
```
GET /api/v1/policies/{policy_number}
```

---

## 7. Developer Docs

**Route**: `/docs` and sub-routes

**Purpose**: Human-friendly API documentation with examples.

**Primary Audience**: Developers integrating with the API.

### Layout

```
+------------------------------------------------------------------+
|  Navigation Bar                                                   |
+------------------------------------------------------------------+
|              |                                                    |
|  [Sidebar]   |  [Main Content]                                   |
|              |                                                    |
|  Overview    |  # Products API                                   |
|              |                                                    |
|  Products    |  The Products API provides access to available    |
|  - List      |  insurance products in the catalog.               |
|  - Get       |                                                    |
|              |  ## List Products                                  |
|  Quotes      |                                                    |
|  - Create    |  Returns all available insurance products.        |
|  - Get       |                                                    |
|              |  ```                                               |
|  Applications|  GET /api/v1/products                             |
|  - Create    |  ```                                               |
|  - Get       |                                                    |
|  - Update    |  ### Response                                      |
|  - Submit    |                                                    |
|              |  ```json                                           |
|  Underwriting|  [                                                 |
|  - List Cases|    {                                               |
|  - Get Case  |      "id": "01JEGXYZ...",                         |
|  - Decide    |      "slug": "term-life-10",                      |
|              |      "name": "10-Year Term Life",                 |
|  Offers      |      "term_years": 10,                            |
|  - Generate  |      "min_coverage": 50000,                       |
|  - Get       |      "max_coverage": 500000,                      |
|  - Accept    |      "base_rate": 0.25                            |
|  - Decline   |    }                                               |
|              |  ]                                                 |
|  Policies    |  ```                                               |
|  - List      |                                                    |
|  - Get       |  ### Code Examples                                 |
|              |                                                    |
|  Errors      |  [curl] [Node.js] [Go]                            |
|              |                                                    |
+------------------------------------------------------------------+
```

### Documentation Structure

#### Overview (`/docs`)

```markdown
# Go Insurance API

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

This demo API does not require authentication.

## Content Type

All requests and responses use `application/json`.

## ID Format

All resource IDs use ULID format (e.g., `01JEH1ABC...`).
Policy numbers use human-readable format: `POL-YYYY-NNNNNN`.

## Lifecycle

```
Products → Quotes → Applications → Underwriting → Offers → Policies
```

## Error Responses

Errors follow RFC 7807 Problem Details format:

```json
{
  "type": "about:blank",
  "title": "Not Found",
  "status": 404,
  "detail": "Quote not found"
}
```

Common status codes:
- 400 Bad Request - Validation error
- 404 Not Found - Resource not found
- 409 Conflict - State conflict (e.g., already submitted)
- 500 Internal Server Error - Server error
```

#### Products (`/docs/products`)

```markdown
# Products API

Products represent insurance offerings with defined coverage limits and pricing.

## List Products

```http
GET /api/v1/products
```

### Response

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique product ID (ULID) |
| slug | string | URL-friendly identifier |
| name | string | Display name |
| term_years | integer | Policy term length |
| min_coverage | integer | Minimum coverage amount |
| max_coverage | integer | Maximum coverage amount |
| base_rate | number | Base monthly rate per $1,000 |

### Examples

**curl**
```bash
curl http://localhost:8080/api/v1/products
```

**Node.js**
```typescript
const response = await fetch('http://localhost:8080/api/v1/products');
const products = await response.json();
```

**Go**
```go
resp, err := http.Get("http://localhost:8080/api/v1/products")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

var products []Product
json.NewDecoder(resp.Body).Decode(&products)
```

---

## Get Product

```http
GET /api/v1/products/{product_slug}
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| product_slug | string | Product slug (e.g., "term-life-10") |

### Examples

**curl**
```bash
curl http://localhost:8080/api/v1/products/term-life-10
```

### Error Responses

**404 Not Found** - Product does not exist
```json
{
  "type": "about:blank",
  "title": "Not Found",
  "status": 404,
  "detail": "Product not found"
}
```
```

#### Quotes (`/docs/quotes`)

```markdown
# Quotes API

Quotes provide pricing for specific coverage configurations.

## Create Quote

```http
POST /api/v1/quotes
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| product_slug | string | Yes | Product to quote |
| coverage_amount | integer | Yes | Coverage amount in dollars |
| term_years | integer | Yes | Policy term |
| age | integer | Yes | Applicant age (18-120) |
| smoker | boolean | No | Smoker status (default: false) |

### Example Request

```json
{
  "product_slug": "term-life-10",
  "coverage_amount": 150000,
  "term_years": 10,
  "age": 35,
  "smoker": false
}
```

### Response

```json
{
  "id": "01JEH1ABC...",
  "product_id": "01JEGXYZ...",
  "product_slug": "term-life-10",
  "coverage_amount": 150000,
  "term_years": 10,
  "monthly_premium": 37.50,
  "status": "priced",
  "created_at": "2025-12-06T23:15:00Z",
  "expires_at": "2025-12-07T23:15:00Z"
}
```

Quotes expire after 24 hours.

### Examples

**curl**
```bash
curl -X POST http://localhost:8080/api/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "product_slug": "term-life-10",
    "coverage_amount": 150000,
    "term_years": 10,
    "age": 35,
    "smoker": false
  }'
```

**Node.js**
```typescript
const response = await fetch('http://localhost:8080/api/v1/quotes', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    product_slug: 'term-life-10',
    coverage_amount: 150000,
    term_years: 10,
    age: 35,
    smoker: false
  })
});
const quote = await response.json();
console.log(`Monthly premium: $${quote.monthly_premium}`);
```

### Error Responses

**400 Bad Request** - Validation error
```json
{
  "type": "about:blank",
  "title": "Bad Request",
  "status": 400,
  "detail": "coverage must be > 0"
}
```

**404 Not Found** - Product not found
```json
{
  "type": "about:blank",
  "title": "Not Found",
  "status": 404,
  "detail": "Product not found"
}
```
```

*(Similar documentation for Applications, Underwriting, Offers, Policies)*

---

## 8. Technical Overview

**Route**: `/tech`

**Purpose**: Explain the system architecture for technical evaluators.

**Primary Audience**: Senior engineers, architects, technical decision-makers.

### Content

```markdown
# Technical Overview

## Architecture

The Go Insurance API is a monolithic REST service built with Go, designed for
reliability, performance, and operational simplicity.

### Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| HTTP Router | Chi |
| Database | DynamoDB (or MongoDB) |
| ID Generation | ULID |
| Documentation | OpenAPI 2.0 (Swagger) |

### System Design

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  HTTP API   │────▶│  DynamoDB   │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Background │
                    │   Workers   │
                    └─────────────┘
```

### Request Flow

1. Client sends HTTP request
2. Chi router matches endpoint
3. Handler validates input
4. Service executes business logic
5. Repository persists to DynamoDB
6. JSON response returned

### Background Workers

Two async workers poll for work every 5 seconds:

**Underwriting Worker**
- Finds applications with status `submitted`
- Runs risk scoring algorithm
- Auto-approves, auto-declines, or refers to manual queue
- Creates underwriting case record
- Updates application status

**Issuance Worker**
- Finds offers with status `accepted`
- Generates policy number (POL-YYYY-NNNNNN)
- Creates policy record
- Sets effective/expiry dates

### Data Model

```
Product (catalog)
    │
    ▼
Quote (pricing, 24h validity)
    │
    ▼
Application (applicant info)
    │
    ├──▶ UnderwritingCase (risk assessment)
    │
    ▼
Offer (binding terms, 30d validity)
    │
    ▼
Policy (issued coverage)
```

### Risk Scoring

The underwriting engine calculates a 0-100 risk score:

| Factor | Points |
|--------|--------|
| Age 51-60 | +25 |
| Age 61-65 | +35 |
| Age 66-80 | +50 |
| Age > 80 | Auto-decline |
| Smoker | +25 |
| Coverage $100k-$250k | +10 |
| Coverage $250k-$500k | +15 |
| Coverage > $500k | +25 |

**Decision Rules**:
- Score ≤ 20 → Auto-approve
- Age < 45 AND non-smoker AND coverage < $250k AND score ≤ 30 → Auto-approve
- Age > 80 → Auto-decline
- Otherwise → Refer to manual review

### Error Handling

All errors use RFC 7807 Problem Details:

```json
{
  "type": "about:blank",
  "title": "Conflict",
  "status": 409,
  "detail": "Application not in draft status"
}
```

### Performance Characteristics

| Metric | Target |
|--------|--------|
| API Response Time | < 100ms p99 |
| Worker Poll Interval | 5 seconds |
| Quote Validity | 24 hours |
| Offer Validity | 30 days |

### Deployment

**Local Development**:
```bash
docker compose up -d dynamodb
go run cmd/api/main.go
go run cmd/seed/main.go
```

**AWS Production**:
- API runs on ECS/EKS or Lambda
- DynamoDB with on-demand capacity
- No infrastructure changes needed

### API Versioning

- Current version: v1
- Base path: `/api/v1`
- Breaking changes will increment version
```

---

## 9. Component Structure

### Recommended Next.js Project Structure

```
src/
├── app/
│   ├── layout.tsx              # Root layout with nav
│   ├── page.tsx                # Landing page
│   ├── demo/
│   │   ├── journey/
│   │   │   └── page.tsx        # Guided journey
│   │   ├── playground/
│   │   │   └── page.tsx        # API playground
│   │   ├── underwriting/
│   │   │   └── page.tsx        # UW workbench
│   │   └── policies/
│   │       └── page.tsx        # Policy dashboard
│   ├── docs/
│   │   ├── page.tsx            # Docs overview
│   │   ├── products/
│   │   ├── quotes/
│   │   ├── applications/
│   │   ├── underwriting/
│   │   ├── offers/
│   │   └── policies/
│   └── tech/
│       └── page.tsx            # Technical overview
├── components/
│   ├── ui/                     # shadcn components
│   ├── layout/
│   │   ├── Navigation.tsx
│   │   └── Footer.tsx
│   ├── journey/
│   │   ├── Stepper.tsx
│   │   ├── ProductSelector.tsx
│   │   ├── QuoteForm.tsx
│   │   ├── ApplicationForm.tsx
│   │   ├── UnderwritingStatus.tsx
│   │   ├── OfferCard.tsx
│   │   ├── PolicyCertificate.tsx
│   │   └── IdTracker.tsx
│   ├── playground/
│   │   ├── ScenarioList.tsx
│   │   ├── RequestBuilder.tsx
│   │   ├── ResponseViewer.tsx
│   │   └── JsonEditor.tsx
│   ├── underwriting/
│   │   ├── CaseQueue.tsx
│   │   ├── CaseDetails.tsx
│   │   └── DecisionForm.tsx
│   ├── policies/
│   │   ├── PolicyTable.tsx
│   │   ├── PolicyFilters.tsx
│   │   └── PolicyDetails.tsx
│   └── docs/
│       ├── DocsSidebar.tsx
│       ├── EndpointDoc.tsx
│       └── CodeExample.tsx
├── lib/
│   ├── api/
│   │   ├── client.ts           # Base API client
│   │   ├── products.ts
│   │   ├── quotes.ts
│   │   ├── applications.ts
│   │   ├── underwriting.ts
│   │   ├── offers.ts
│   │   └── policies.ts
│   └── types/
│       └── api.ts              # TypeScript types from Swagger
└── hooks/
    ├── useProducts.ts
    ├── useQuote.ts
    ├── useApplication.ts
    ├── useUnderwriting.ts
    ├── useOffer.ts
    └── usePolicies.ts
```

---

## 10. API Integration Layer

### TypeScript Types (`lib/types/api.ts`)

```typescript
// Products
export interface Product {
  id: string;
  slug: string;
  name: string;
  term_years: number;
  min_coverage: number;
  max_coverage: number;
  base_rate: number;
}

// Quotes
export interface QuoteInput {
  product_slug: string;
  coverage_amount: number;
  term_years: number;
  age: number;
  smoker?: boolean;
}

export interface Quote {
  id: string;
  product_id: string;
  product_slug: string;
  coverage_amount: number;
  term_years: number;
  monthly_premium: number;
  status: 'new' | 'priced' | 'expired';
  created_at: string;
  expires_at: string;
}

// Applications
export interface Applicant {
  first_name: string;
  last_name: string;
  email: string;
  date_of_birth: string;
  age: number;
  smoker: boolean;
  state: string;
}

export interface ApplicationInput {
  quote_id: string;
  applicant: Applicant;
}

export interface ApplicationPatch {
  applicant?: Partial<Applicant>;
}

export interface Application {
  id: string;
  quote_id: string;
  product_id: string;
  product_slug: string;
  coverage_amount: number;
  term_years: number;
  monthly_premium: number;
  applicant: Applicant;
  status: 'draft' | 'submitted' | 'under_review' | 'approved' | 'declined';
  created_at: string;
  updated_at: string;
  submitted_at?: string;
}

// Underwriting
export interface RiskFactors {
  age: number;
  smoker: boolean;
  coverage_amount: number;
  term_years: number;
}

export interface RiskScore {
  score: number;
  flags: string[];
  recommended: 'approved' | 'declined' | 'referred';
}

export interface UnderwritingCase {
  id: string;
  application_id: string;
  risk_factors: RiskFactors;
  risk_score: RiskScore;
  decision: 'pending' | 'approved' | 'declined' | 'referred';
  method: 'auto' | 'manual';
  decided_by: string;
  reason: string;
  created_at: string;
  updated_at: string;
  decided_at?: string;
}

export interface UWDecisionInput {
  decision: 'approved' | 'declined';
  reason: string;
}

// Offers
export interface Offer {
  id: string;
  application_id: string;
  product_slug: string;
  coverage_amount: number;
  term_years: number;
  monthly_premium: number;
  status: 'pending' | 'accepted' | 'declined' | 'expired' | 'issued';
  created_at: string;
  expires_at: string;
  accepted_at?: string;
  declined_at?: string;
}

// Policies
export interface Policy {
  id: string;
  number: string;
  application_id: string;
  offer_id: string;
  product_slug: string;
  coverage_amount: number;
  term_years: number;
  monthly_premium: number;
  insured: Applicant;
  status: 'active' | 'lapsed' | 'cancelled' | 'expired';
  effective_date: string;
  expiry_date: string;
  issued_at: string;
}

export interface PolicyList {
  items: Policy[];
  total: number;
  limit: number;
  offset: number;
}

export interface PolicyFilter {
  application_id?: string;
  status?: Policy['status'];
  limit?: number;
  offset?: number;
}

// Errors
export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
}
```

### API Client (`lib/api/client.ts`)

```typescript
const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export class ApiError extends Error {
  constructor(
    public status: number,
    public problem: ProblemDetails
  ) {
    super(problem.detail);
    this.name = 'ApiError';
  }
}

export async function apiRequest<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const url = `${BASE_URL}${path}`;

  const response = await fetch(url, {
    method,
    headers: {
      'Content-Type': 'application/json',
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    const problem: ProblemDetails = await response.json();
    throw new ApiError(response.status, problem);
  }

  return response.json();
}

export const api = {
  get: <T>(path: string) => apiRequest<T>('GET', path),
  post: <T>(path: string, body?: unknown) => apiRequest<T>('POST', path, body),
  patch: <T>(path: string, body: unknown) => apiRequest<T>('PATCH', path, body),
};
```

### Example Hook (`hooks/useQuote.ts`)

```typescript
import { useMutation, useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api/client';
import type { Quote, QuoteInput } from '@/lib/types/api';

export function useCreateQuote() {
  return useMutation({
    mutationFn: (input: QuoteInput) =>
      api.post<Quote>('/quotes', input),
  });
}

export function useQuote(id: string | undefined) {
  return useQuery({
    queryKey: ['quote', id],
    queryFn: () => api.get<Quote>(`/quotes/${id}`),
    enabled: !!id,
  });
}
```

---

## Summary

This blueprint provides a complete design for a frontend tech demo that:

1. **Educates** non-technical users about the insurance API lifecycle
2. **Enables** developers to explore and test all endpoints
3. **Demonstrates** real-world workflows like underwriting and policy management
4. **Documents** the API with examples in multiple languages
5. **Explains** the technical architecture for evaluators

The design maps directly to the Swagger spec, using only the defined endpoints and schemas. All example requests and responses are valid for the actual API.

Next steps:
1. Initialize Next.js project with TypeScript
2. Install dependencies (shadcn/ui, TanStack Query, Monaco Editor)
3. Implement API client layer
4. Build pages following this blueprint
5. Connect to running API at localhost:8080
