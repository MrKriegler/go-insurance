import type {
  Product,
  Quote,
  QuoteInput,
  Application,
  ApplicationInput,
  ApplicationPatch,
  UnderwritingCase,
  UWDecisionInput,
  Offer,
  Policy,
  PolicyList,
  PolicyFilter,
  ProblemDetails,
} from "./types";

// Use full URL in production, proxy in development
const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";
const API_KEY = process.env.NEXT_PUBLIC_API_KEY || "demo-api-key-12345";

export class ApiError extends Error {
  constructor(
    public status: number,
    public problem: ProblemDetails
  ) {
    super(problem.detail);
    this.name = "ApiError";
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const url = `${BASE_URL}${path}`;

  const response = await fetch(url, {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": API_KEY,
    },
    body: body ? JSON.stringify(body) : undefined,
  });

  const data = await response.json();

  if (!response.ok) {
    throw new ApiError(response.status, data as ProblemDetails);
  }

  return data as T;
}

// Products API
export const products = {
  list: () => request<Product[]>("GET", "/products"),
  get: (slug: string) => request<Product>("GET", `/products/${slug}`),
};

// Quotes API
export const quotes = {
  create: (input: QuoteInput) => request<Quote>("POST", "/quotes", input),
  get: (id: string) => request<Quote>("GET", `/quotes/${id}`),
};

// Applications API
export const applications = {
  create: (input: ApplicationInput) =>
    request<Application>("POST", "/applications", input),
  get: (id: string) => request<Application>("GET", `/applications/${id}`),
  patch: (id: string, input: ApplicationPatch) =>
    request<Application>("PATCH", `/applications/${id}`, input),
  submit: (id: string) =>
    request<Application>("POST", `/applications/${id}:submit`),
};

// Underwriting API
export const underwriting = {
  listCases: () => request<UnderwritingCase[]>("GET", "/underwriting/cases"),
  getCase: (id: string) =>
    request<UnderwritingCase>("GET", `/underwriting/cases/${id}`),
  decide: (id: string, input: UWDecisionInput) =>
    request<UnderwritingCase>("POST", `/underwriting/cases/${id}:decide`, input),
};

// Offers API
export const offers = {
  create: (applicationId: string) =>
    request<Offer>("POST", `/applications/${applicationId}/offers`),
  get: (id: string) => request<Offer>("GET", `/offers/${id}`),
  accept: (id: string) => request<Offer>("POST", `/offers/${id}:accept`),
  decline: (id: string) => request<Offer>("POST", `/offers/${id}:decline`),
};

// Policies API
export const policies = {
  list: (filter?: PolicyFilter) => {
    const params = new URLSearchParams();
    if (filter?.application_id) params.set("application_id", filter.application_id);
    if (filter?.status) params.set("status", filter.status);
    if (filter?.limit) params.set("limit", String(filter.limit));
    if (filter?.offset) params.set("offset", String(filter.offset));
    const query = params.toString();
    return request<PolicyList>("GET", `/policies${query ? `?${query}` : ""}`);
  },
  get: (number: string) => request<Policy>("GET", `/policies/${number}`),
};

// Export all as single object
export const api = {
  products,
  quotes,
  applications,
  underwriting,
  offers,
  policies,
};
