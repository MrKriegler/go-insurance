"use client";

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import { formatCurrency, truncateId } from "@/lib/utils";
import type { Product, Quote, Application, UnderwritingCase, Offer, Policy } from "@/lib/types";

type Step = 1 | 2 | 3 | 4 | 5 | 6 | 7;

const STEPS = [
  { num: 1, label: "Products" },
  { num: 2, label: "Quote" },
  { num: 3, label: "Application" },
  { num: 4, label: "Submit" },
  { num: 5, label: "Underwriting" },
  { num: 6, label: "Offer" },
  { num: 7, label: "Policy" },
];

export default function JourneyPage() {
  const [step, setStep] = useState<Step>(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [apiLog, setApiLog] = useState<{ method: string; path: string; request?: object; response?: object; status?: number } | null>(null);

  const [products, setProducts] = useState<Product[]>([]);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [quote, setQuote] = useState<Quote | null>(null);
  const [application, setApplication] = useState<Application | null>(null);
  const [uwCase, setUwCase] = useState<UnderwritingCase | null>(null);
  const [offer, setOffer] = useState<Offer | null>(null);
  const [policy, setPolicy] = useState<Policy | null>(null);

  const [quoteForm, setQuoteForm] = useState({ coverage: 150000, age: 35, smoker: false });
  const [applicantForm, setApplicantForm] = useState({
    first_name: "John",
    last_name: "Doe",
    email: "john.doe@example.com",
    date_of_birth: "1989-06-15",
    state: "CA",
  });
  const [formErrors, setFormErrors] = useState<string[]>([]);

  const logApi = (method: string, path: string, request?: object, response?: object, status?: number) => {
    setApiLog({ method, path, request, response, status });
  };

  const validateQuoteForm = (): string[] => {
    const errors: string[] = [];
    if (!selectedProduct) return ["No product selected"];
    if (quoteForm.coverage < selectedProduct.min_coverage || quoteForm.coverage > selectedProduct.max_coverage) {
      errors.push(`Coverage must be between ${formatCurrency(selectedProduct.min_coverage)} and ${formatCurrency(selectedProduct.max_coverage)}`);
    }
    if (quoteForm.age < 18 || quoteForm.age > 120) {
      errors.push("Age must be between 18 and 120");
    }
    return errors;
  };

  const validateApplicantForm = (): string[] => {
    const errors: string[] = [];
    if (!applicantForm.first_name.trim()) errors.push("First name is required");
    if (!applicantForm.last_name.trim()) errors.push("Last name is required");
    if (!applicantForm.email.trim()) {
      errors.push("Email is required");
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(applicantForm.email)) {
      errors.push("Invalid email format");
    }
    if (!applicantForm.date_of_birth) errors.push("Date of birth is required");
    if (!applicantForm.state.trim()) {
      errors.push("State is required");
    } else if (applicantForm.state.length !== 2) {
      errors.push("State must be a 2-letter code");
    }
    return errors;
  };

  const loadProducts = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.products.list();
      setProducts(data);
      logApi("GET", "/api/v1/products", undefined, data, 200);
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
      logApi("GET", "/api/v1/products", undefined, err.problem, err.status);
    } finally {
      setLoading(false);
    }
  };

  const createQuote = async () => {
    if (!selectedProduct) return;
    const errors = validateQuoteForm();
    if (errors.length > 0) {
      setFormErrors(errors);
      return;
    }
    setFormErrors([]);
    setLoading(true);
    setError(null);
    const input = {
      product_slug: selectedProduct.slug,
      coverage_amount: quoteForm.coverage,
      term_years: selectedProduct.term_years,
      age: quoteForm.age,
      smoker: quoteForm.smoker,
    };
    try {
      const data = await api.quotes.create(input);
      setQuote(data);
      logApi("POST", "/api/v1/quotes", input, data, 201);
      setStep(3);
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
      logApi("POST", "/api/v1/quotes", input, err.problem, err.status);
    } finally {
      setLoading(false);
    }
  };

  const createApplication = async () => {
    if (!quote) return;
    const errors = validateApplicantForm();
    if (errors.length > 0) {
      setFormErrors(errors);
      return;
    }
    setFormErrors([]);
    setLoading(true);
    setError(null);
    const input = {
      quote_id: quote.id,
      applicant: { ...applicantForm, age: quoteForm.age, smoker: quoteForm.smoker },
    };
    try {
      const data = await api.applications.create(input);
      setApplication(data);
      logApi("POST", "/api/v1/applications", input, data, 201);
      setStep(4);
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
      logApi("POST", "/api/v1/applications", input, err.problem, err.status);
    } finally {
      setLoading(false);
    }
  };

  const submitApplication = async () => {
    if (!application) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.applications.submit(application.id);
      setApplication(data);
      logApi("POST", `/api/v1/applications/${application.id}:submit`, undefined, data, 200);
      setStep(5);
      pollUnderwriting(application.id);
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
      logApi("POST", `/api/v1/applications/${application.id}:submit`, undefined, err.problem, err.status);
    } finally {
      setLoading(false);
    }
  };

  const pollUnderwriting = async (appId: string, existingCaseId?: string) => {
    setLoading(true);
    let attempts = 0;
    const poll = async () => {
      attempts++;
      try {
        const appData = await api.applications.get(appId);
        setApplication(appData);
        logApi("GET", `/api/v1/applications/${appId}`, undefined, appData, 200);

        // Check if we have a final decision
        if (appData.status === "approved" || appData.status === "declined") {
          // If we have a case ID, fetch it directly (it won't be in listCases after decision)
          if (existingCaseId) {
            const uw = await api.underwriting.getCase(existingCaseId);
            setUwCase(uw);
            logApi("GET", `/api/v1/underwriting/cases/${existingCaseId}`, undefined, uw, 200);
          }
          setLoading(false);
          if (appData.status === "approved") setStep(6);
          return;
        }

        // Check if case is referred (application stays in under_review)
        if (appData.status === "under_review") {
          const cases = await api.underwriting.listCases();
          const uw = cases.find((c) => c.application_id === appId);
          if (uw && uw.decision === "referred") {
            setUwCase(uw);
            logApi("GET", "/api/v1/underwriting/cases", undefined, cases, 200);
            setLoading(false);
            return; // Stop polling - manual review needed
          }
        }

        if (attempts < 15) setTimeout(poll, 2000);
        else {
          setLoading(false);
          setError("Underwriting is taking longer than expected.");
        }
      } catch (e) {
        setLoading(false);
        setError((e as ApiError).message);
      }
    };
    poll();
  };

  const generateOffer = async () => {
    if (!application) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.offers.create(application.id);
      setOffer(data);
      logApi("POST", `/api/v1/applications/${application.id}/offers`, undefined, data, 201);
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
      logApi("POST", `/api/v1/applications/${application.id}/offers`, undefined, err.problem, err.status);
    } finally {
      setLoading(false);
    }
  };

  const acceptOffer = async () => {
    if (!offer) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.offers.accept(offer.id);
      setOffer(data);
      logApi("POST", `/api/v1/offers/${offer.id}:accept`, undefined, data, 200);
      setStep(7);
      pollPolicy(); // pollPolicy manages its own loading state
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
      logApi("POST", `/api/v1/offers/${offer.id}:accept`, undefined, err.problem, err.status);
      setLoading(false);
    }
  };

  const pollPolicy = async () => {
    setLoading(true);
    let attempts = 0;
    const poll = async () => {
      attempts++;
      try {
        const data = await api.policies.list({ application_id: application?.id });
        logApi("GET", `/api/v1/policies?application_id=${application?.id}`, undefined, data, 200);
        if (data.items && data.items.length > 0) {
          setPolicy(data.items[0]);
          setLoading(false);
          return;
        }
        if (attempts < 15) setTimeout(poll, 2000);
        else {
          setLoading(false);
          setError("Policy issuance is taking longer than expected.");
        }
      } catch (e) {
        setLoading(false);
        setError((e as ApiError).message);
      }
    };
    poll();
  };

  const renderStepContent = () => {
    switch (step) {
      case 1:
        return (
          <div className="space-y-6">
            <p className="text-muted-foreground">Select an insurance product to begin.</p>
            {products.length === 0 ? (
              <button
                onClick={loadProducts}
                disabled={loading}
                className="h-10 px-6 text-sm font-medium bg-white text-black rounded hover:bg-white/90 disabled:opacity-50 transition-colors"
              >
                {loading ? "Loading..." : "Load Products"}
              </button>
            ) : (
              <div className="space-y-3">
                {products.map((p) => (
                  <div
                    key={p.id}
                    onClick={() => setSelectedProduct(p)}
                    className={`p-4 border rounded cursor-pointer transition-all ${
                      selectedProduct?.id === p.id
                        ? "border-emerald-500 bg-emerald-500/10"
                        : "border-border hover:border-muted-foreground/50"
                    }`}
                  >
                    <div className="flex justify-between items-start">
                      <div>
                        <h3 className="font-semibold">{p.name}</h3>
                        <p className="text-sm text-muted-foreground">{p.term_years} year term</p>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        ${p.base_rate}/mo per $1k
                      </span>
                    </div>
                    <div className="mt-2 text-xs text-muted-foreground">
                      Coverage: {formatCurrency(p.min_coverage)} - {formatCurrency(p.max_coverage)}
                    </div>
                  </div>
                ))}
                {selectedProduct && (
                  <button
                    onClick={() => setStep(2)}
                    className="w-full h-10 text-sm font-medium bg-white text-black rounded hover:bg-white/90 transition-colors"
                  >
                    Continue with {selectedProduct.name}
                  </button>
                )}
              </div>
            )}
          </div>
        );

      case 2:
        return (
          <div className="space-y-6">
            <p className="text-muted-foreground">Configure coverage and get a quote.</p>
            <div className="space-y-4">
              <div>
                <label className="text-sm text-muted-foreground block mb-2">Coverage Amount</label>
                <input
                  type="number"
                  value={quoteForm.coverage}
                  onChange={(e) => setQuoteForm({ ...quoteForm, coverage: parseInt(e.target.value) || 0 })}
                  className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  Range: {formatCurrency(selectedProduct?.min_coverage || 0)} - {formatCurrency(selectedProduct?.max_coverage || 0)}
                </p>
              </div>
              <div>
                <label className="text-sm text-muted-foreground block mb-2">Age</label>
                <input
                  type="number"
                  value={quoteForm.age}
                  onChange={(e) => setQuoteForm({ ...quoteForm, age: parseInt(e.target.value) || 18 })}
                  className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground"
                />
              </div>
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={quoteForm.smoker}
                  onChange={(e) => setQuoteForm({ ...quoteForm, smoker: e.target.checked })}
                  className="w-4 h-4 rounded border-border bg-secondary"
                />
                <span className="text-sm">Smoker</span>
              </label>
              <button
                onClick={createQuote}
                disabled={loading}
                className="w-full h-10 text-sm font-medium bg-white text-black rounded hover:bg-white/90 disabled:opacity-50 transition-colors"
              >
                {loading ? "Getting Quote..." : "Get Quote"}
              </button>
            </div>
          </div>
        );

      case 3:
        return (
          <div className="space-y-6">
            <div className="p-4 bg-emerald-500/10 border border-emerald-500/30 rounded">
              <div className="text-2xl font-bold">{formatCurrency(quote?.monthly_premium || 0)}/mo</div>
              <div className="text-sm text-muted-foreground">
                {formatCurrency(quote?.coverage_amount || 0)} for {quote?.term_years} years
              </div>
            </div>
            <p className="text-muted-foreground">Enter applicant information.</p>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm text-muted-foreground block mb-2">First Name</label>
                <input
                  value={applicantForm.first_name}
                  onChange={(e) => setApplicantForm({ ...applicantForm, first_name: e.target.value })}
                  className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground"
                />
              </div>
              <div>
                <label className="text-sm text-muted-foreground block mb-2">Last Name</label>
                <input
                  value={applicantForm.last_name}
                  onChange={(e) => setApplicantForm({ ...applicantForm, last_name: e.target.value })}
                  className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground"
                />
              </div>
            </div>
            <div>
              <label className="text-sm text-muted-foreground block mb-2">Email</label>
              <input
                type="email"
                value={applicantForm.email}
                onChange={(e) => setApplicantForm({ ...applicantForm, email: e.target.value })}
                className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm text-muted-foreground block mb-2">Date of Birth</label>
                <input
                  type="date"
                  value={applicantForm.date_of_birth}
                  onChange={(e) => setApplicantForm({ ...applicantForm, date_of_birth: e.target.value })}
                  className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground"
                />
              </div>
              <div>
                <label className="text-sm text-muted-foreground block mb-2">State</label>
                <input
                  value={applicantForm.state}
                  onChange={(e) => setApplicantForm({ ...applicantForm, state: e.target.value })}
                  maxLength={2}
                  className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground uppercase"
                />
              </div>
            </div>
            <button
              onClick={createApplication}
              disabled={loading}
              className="w-full h-10 text-sm font-medium bg-white text-black rounded hover:bg-white/90 disabled:opacity-50 transition-colors"
            >
              {loading ? "Creating..." : "Create Application"}
            </button>
          </div>
        );

      case 4:
        return (
          <div className="space-y-6">
            <div className="p-4 bg-secondary border border-border rounded">
              <div className="flex items-center justify-between">
                <span className="font-semibold">Application Created</span>
                <span className="text-xs px-2 py-1 bg-muted rounded">{application?.status}</span>
              </div>
            </div>
            <p className="text-muted-foreground">Submit for underwriting review.</p>
            <button
              onClick={submitApplication}
              disabled={loading}
              className="w-full h-10 text-sm font-medium bg-white text-black rounded hover:bg-white/90 disabled:opacity-50 transition-colors"
            >
              {loading ? "Submitting..." : "Submit for Underwriting"}
            </button>
          </div>
        );

      case 5:
        return (
          <div className="space-y-6">
            {loading ? (
              <div className="py-12 text-center">
                <div className="inline-block w-6 h-6 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                <p className="text-muted-foreground mt-4">Processing underwriting...</p>
              </div>
            ) : uwCase ? (
              <>
                <div className={`p-4 rounded border ${
                  uwCase.decision === "approved" ? "bg-emerald-500/10 border-emerald-500/30" :
                  uwCase.decision === "declined" ? "bg-red-500/10 border-red-500/30" :
                  "bg-yellow-500/10 border-yellow-500/30"
                }`}>
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-semibold">{uwCase.decision.toUpperCase()}</span>
                    <span className="text-xs px-2 py-1 bg-background/50 rounded">{uwCase.method}</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {uwCase.decision === "referred"
                      ? "This application requires manual review by an underwriter."
                      : uwCase.reason}
                  </p>
                </div>
                <div className="p-4 bg-secondary border border-border rounded">
                  <div className="flex justify-between items-center mb-3">
                    <span className="text-sm text-muted-foreground">Risk Score</span>
                    <span className="font-bold">{uwCase.risk_score.score}/100</span>
                  </div>
                  <div className="h-2 bg-background rounded-full overflow-hidden">
                    <div
                      className={`h-full transition-all ${
                        uwCase.risk_score.score <= 30 ? "bg-emerald-500" :
                        uwCase.risk_score.score <= 60 ? "bg-yellow-500" : "bg-red-500"
                      }`}
                      style={{ width: `${uwCase.risk_score.score}%` }}
                    />
                  </div>
                  {uwCase.risk_score.flags && uwCase.risk_score.flags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-3">
                      {uwCase.risk_score.flags.map((flag) => (
                        <span key={flag} className="text-xs px-2 py-1 bg-background rounded">{flag}</span>
                      ))}
                    </div>
                  )}
                </div>
                {uwCase.decision === "approved" && (
                  <button
                    onClick={() => setStep(6)}
                    className="w-full h-10 text-sm font-medium bg-white text-black rounded hover:bg-white/90 transition-colors"
                  >
                    Continue to Offer
                  </button>
                )}
                {uwCase.decision === "referred" && (
                  <div className="space-y-3">
                    <p className="text-sm text-muted-foreground">
                      Go to the Underwriting page to approve or decline this case, then return here.
                    </p>
                    <div className="flex gap-3">
                      <a
                        href="/demo/underwriting"
                        target="_blank"
                        className="flex-1 h-10 text-sm font-medium bg-yellow-500 text-black rounded hover:bg-yellow-400 transition-colors flex items-center justify-center"
                      >
                        Open Underwriting
                      </a>
                      <button
                        onClick={() => pollUnderwriting(application!.id, uwCase.id)}
                        disabled={loading}
                        className="flex-1 h-10 text-sm font-medium border border-border rounded hover:bg-secondary disabled:opacity-50 transition-colors"
                      >
                        {loading ? "Checking..." : "Check Status"}
                      </button>
                    </div>
                  </div>
                )}
                {uwCase.decision === "declined" && (
                  <div className="p-4 bg-red-500/10 border border-red-500/30 rounded text-center">
                    <p className="text-red-400">Application was declined.</p>
                  </div>
                )}
              </>
            ) : (
              <div className="py-12 text-center text-muted-foreground">
                <p>Waiting for underwriting result...</p>
              </div>
            )}
          </div>
        );

      case 6:
        return (
          <div className="space-y-6">
            {!offer ? (
              <>
                <p className="text-muted-foreground">Application approved. Generate an offer.</p>
                <button
                  onClick={generateOffer}
                  disabled={loading}
                  className="w-full h-10 text-sm font-medium bg-white text-black rounded hover:bg-white/90 disabled:opacity-50 transition-colors"
                >
                  {loading ? "Generating..." : "Generate Offer"}
                </button>
              </>
            ) : (
              <>
                <div className="p-6 bg-secondary border border-border rounded text-center">
                  <div className="text-xs text-muted-foreground mb-1">Monthly Premium</div>
                  <div className="text-3xl font-bold">{formatCurrency(offer.monthly_premium)}</div>
                  <div className="text-sm text-muted-foreground mt-2">
                    {formatCurrency(offer.coverage_amount)} for {offer.term_years} years
                  </div>
                  <div className="text-xs text-muted-foreground mt-4">
                    Expires: {new Date(offer.expires_at).toLocaleDateString()}
                  </div>
                </div>
                {offer.status === "pending" && (
                  <div className="flex gap-3">
                    <button
                      onClick={acceptOffer}
                      disabled={loading}
                      className="flex-1 h-10 text-sm font-medium bg-emerald-500 text-white rounded hover:bg-emerald-600 disabled:opacity-50 transition-colors"
                    >
                      {loading ? "Accepting..." : "Accept Offer"}
                    </button>
                    <button className="flex-1 h-10 text-sm font-medium border border-border rounded hover:bg-secondary transition-colors">
                      Decline
                    </button>
                  </div>
                )}
              </>
            )}
          </div>
        );

      case 7:
        return (
          <div className="space-y-6">
            {policy ? (
              <>
                <div className="p-6 bg-emerald-500/10 border border-emerald-500/30 rounded text-center">
                  <div className="text-xs text-emerald-400 mb-1">Policy Number</div>
                  <div className="text-2xl font-bold font-mono">{policy.number}</div>
                </div>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <div className="text-muted-foreground">Insured</div>
                    <div className="font-semibold">{policy.insured.first_name} {policy.insured.last_name}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Status</div>
                    <div className="font-semibold text-emerald-400">{policy.status}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Coverage</div>
                    <div className="font-semibold">{formatCurrency(policy.coverage_amount)}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Premium</div>
                    <div className="font-semibold">{formatCurrency(policy.monthly_premium)}/mo</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Effective</div>
                    <div className="font-semibold">{new Date(policy.effective_date).toLocaleDateString()}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Expires</div>
                    <div className="font-semibold">{new Date(policy.expiry_date).toLocaleDateString()}</div>
                  </div>
                </div>
                <div className="p-4 bg-emerald-500/10 border border-emerald-500/30 rounded text-center">
                  <p className="text-emerald-400">Policy issued successfully.</p>
                </div>
              </>
            ) : (
              <div className="py-12 text-center">
                <div className="inline-block w-6 h-6 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                <p className="text-muted-foreground mt-4">Waiting for policy...</p>
              </div>
            )}
          </div>
        );
    }
  };

  return (
    <div className="min-h-screen">
      {/* Header */}
      <div className="border-b border-border/50 bg-card">
        <div className="max-w-7xl mx-auto px-6 py-8">
          <h1 className="text-3xl font-bold mb-2">Journey Demo</h1>
          <p className="text-muted-foreground">Complete insurance lifecycle from quote to policy.</p>
        </div>
      </div>

      {/* Progress Bar */}
      <div className="border-b border-border/50 bg-secondary/30">
        <div className="max-w-7xl mx-auto px-6">
          <div className="flex items-center gap-1 py-4 overflow-x-auto">
            {STEPS.map((s, i) => (
              <div key={s.num} className="flex items-center">
                <div className="flex items-center gap-2">
                  <div className={`w-7 h-7 rounded flex items-center justify-center text-xs font-medium transition-colors ${
                    step >= s.num ? "bg-white text-black" : "bg-secondary text-muted-foreground"
                  }`}>
                    {s.num}
                  </div>
                  <span className={`text-sm whitespace-nowrap ${step >= s.num ? "text-foreground" : "text-muted-foreground"}`}>
                    {s.label}
                  </span>
                </div>
                {i < STEPS.length - 1 && (
                  <div className={`w-8 h-px mx-3 ${step > s.num ? "bg-white" : "bg-border"}`} />
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-6 py-8">
        {error && (
          <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded text-red-400">
            {error}
          </div>
        )}
        {formErrors.length > 0 && (
          <div className="mb-6 p-4 bg-yellow-500/10 border border-yellow-500/30 rounded text-yellow-400">
            <ul className="list-disc list-inside space-y-1">
              {formErrors.map((e, i) => (
                <li key={i}>{e}</li>
              ))}
            </ul>
          </div>
        )}

        <div className="grid lg:grid-cols-2 gap-8">
          {/* Left Panel */}
          <div className="bg-card border border-border rounded-lg overflow-hidden">
            <div className="px-6 py-4 border-b border-border bg-secondary/30">
              <h2 className="font-semibold">Step {step}: {STEPS[step - 1].label}</h2>
            </div>
            <div className="p-6">
              {renderStepContent()}
            </div>
          </div>

          {/* Right Panel - API Inspector */}
          <div className="bg-card border border-border rounded-lg overflow-hidden">
            <div className="px-6 py-4 border-b border-border bg-secondary/30">
              <h2 className="font-semibold">API Inspector</h2>
            </div>
            <div className="p-6">
              {apiLog ? (
                <div className="space-y-4">
                  <div className="flex items-center gap-3">
                    <span className={`text-xs font-medium px-2 py-1 rounded ${
                      apiLog.status && apiLog.status < 400 ? "bg-emerald-500/20 text-emerald-400" : "bg-red-500/20 text-red-400"
                    }`}>
                      {apiLog.method}
                    </span>
                    <code className="text-sm text-muted-foreground">{apiLog.path}</code>
                    {apiLog.status && (
                      <span className="text-xs text-muted-foreground">{apiLog.status}</span>
                    )}
                  </div>
                  {apiLog.request && (
                    <div>
                      <div className="text-xs text-muted-foreground mb-2">Request</div>
                      <pre className="p-4 bg-secondary rounded text-xs overflow-x-auto">
                        {JSON.stringify(apiLog.request, null, 2)}
                      </pre>
                    </div>
                  )}
                  {apiLog.response && (
                    <div>
                      <div className="text-xs text-muted-foreground mb-2">Response</div>
                      <pre className="p-4 bg-secondary rounded text-xs overflow-x-auto max-h-64 overflow-y-auto">
                        {JSON.stringify(apiLog.response, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              ) : (
                <p className="text-muted-foreground text-sm">API calls will appear here.</p>
              )}
            </div>
          </div>
        </div>

        {/* ID Tracker */}
        <div className="mt-6 p-4 bg-card border border-border rounded-lg">
          <div className="flex flex-wrap gap-6 text-sm">
            {quote && (
              <div>
                <span className="text-muted-foreground">Quote:</span>{" "}
                <code className="text-xs bg-secondary px-2 py-1 rounded">{truncateId(quote.id)}</code>
              </div>
            )}
            {application && (
              <div>
                <span className="text-muted-foreground">Application:</span>{" "}
                <code className="text-xs bg-secondary px-2 py-1 rounded">{truncateId(application.id)}</code>
              </div>
            )}
            {uwCase && (
              <div>
                <span className="text-muted-foreground">UW Case:</span>{" "}
                <code className="text-xs bg-secondary px-2 py-1 rounded">{truncateId(uwCase.id)}</code>
              </div>
            )}
            {offer && (
              <div>
                <span className="text-muted-foreground">Offer:</span>{" "}
                <code className="text-xs bg-secondary px-2 py-1 rounded">{truncateId(offer.id)}</code>
              </div>
            )}
            {policy && (
              <div>
                <span className="text-muted-foreground">Policy:</span>{" "}
                <code className="text-xs bg-emerald-500/20 text-emerald-400 px-2 py-1 rounded font-semibold">{policy.number}</code>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
