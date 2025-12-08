// API Types - All snake_case to match Go JSON serialization

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
  status: "new" | "priced" | "expired";
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
  applicant: {
    first_name: string;
    last_name: string;
    email: string;
    date_of_birth: string;
    age: number;
    smoker: boolean;
    state: string;
  };
}

export interface ApplicationPatch {
  applicant?: Partial<ApplicationInput["applicant"]>;
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
  status: "draft" | "submitted" | "under_review" | "approved" | "declined";
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
  recommended?: "approved" | "declined" | "referred";
}

export interface UnderwritingCase {
  id: string;
  application_id: string;
  risk_factors: RiskFactors;
  risk_score: RiskScore;
  decision: "pending" | "approved" | "declined" | "referred";
  method: "auto" | "manual";
  decided_by: string;
  reason: string;
  created_at: string;
  updated_at: string;
  decided_at?: string;
}

export interface UWDecisionInput {
  decision: "approved" | "declined";
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
  status: "pending" | "accepted" | "declined" | "expired" | "issued";
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
  status: "active" | "lapsed" | "cancelled" | "expired";
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
  status?: Policy["status"];
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
