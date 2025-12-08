"use client";

import { useState, useEffect } from "react";
import { api, ApiError } from "@/lib/api";
import { formatCurrency, truncateId } from "@/lib/utils";
import type { UnderwritingCase } from "@/lib/types";

export default function UnderwritingPage() {
  const [cases, setCases] = useState<UnderwritingCase[]>([]);
  const [selected, setSelected] = useState<UnderwritingCase | null>(null);
  const [loading, setLoading] = useState(false);
  const [deciding, setDeciding] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [decision, setDecision] = useState<"approved" | "declined">("approved");
  const [reason, setReason] = useState("");

  const loadCases = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.underwriting.listCases();
      setCases(data.filter((c) => c.decision === "referred" || c.decision === "pending"));
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCases();
  }, []);

  const submitDecision = async () => {
    if (!selected || !reason) return;
    setDeciding(true);
    setError(null);
    try {
      await api.underwriting.decide(selected.id, { decision, reason });
      setSelected(null);
      setReason("");
      loadCases();
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
    } finally {
      setDeciding(false);
    }
  };

  return (
    <div className="min-h-screen">
      {/* Header */}
      <div className="border-b border-border/50 bg-card">
        <div className="max-w-7xl mx-auto px-6 py-8">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold mb-2">Underwriting</h1>
              <p className="text-muted-foreground">Review and decide on referred cases</p>
            </div>
            <button
              onClick={loadCases}
              disabled={loading}
              className="h-10 px-6 text-sm font-medium border border-border rounded hover:bg-secondary disabled:opacity-50 transition-colors"
            >
              {loading ? "Loading..." : "Refresh Queue"}
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-8">
        {error && (
          <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded text-red-400">
            {error}
          </div>
        )}

        <div className="grid lg:grid-cols-3 gap-8">
          {/* Case Queue */}
          <div className="lg:col-span-1">
            <div className="bg-card border border-border rounded-lg overflow-hidden">
              <div className="px-6 py-4 border-b border-border bg-secondary/30">
                <h2 className="font-semibold">Referred Cases ({cases.length})</h2>
              </div>
              <div className="p-4">
                {loading ? (
                  <div className="py-12 text-center">
                    <div className="inline-block w-6 h-6 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                    <p className="text-muted-foreground mt-4">Loading cases...</p>
                  </div>
                ) : cases.length === 0 ? (
                  <p className="text-muted-foreground text-sm text-center py-12">
                    No cases pending review.
                  </p>
                ) : (
                  <div className="space-y-2">
                    {cases.map((c) => (
                      <div
                        key={c.id}
                        onClick={() => setSelected(c)}
                        className={`p-4 border rounded cursor-pointer transition-all ${
                          selected?.id === c.id
                            ? "border-yellow-500 bg-yellow-500/10"
                            : "border-border hover:border-muted-foreground/50"
                        }`}
                      >
                        <div className="flex justify-between items-start">
                          <div>
                            <div className="font-semibold text-sm">
                              Age {c.risk_factors.age}, {c.risk_factors.smoker ? "Smoker" : "Non-smoker"}
                            </div>
                            <div className="text-xs text-muted-foreground mt-1">
                              {formatCurrency(c.risk_factors.coverage_amount)} / {c.risk_factors.term_years}yr
                            </div>
                          </div>
                          <span className="text-xs font-medium px-2 py-1 bg-yellow-500/20 text-yellow-400 rounded">
                            {c.risk_score.score}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Case Details */}
          <div className="lg:col-span-2">
            {selected ? (
              <div className="bg-card border border-border rounded-lg overflow-hidden">
                <div className="px-6 py-4 border-b border-border bg-secondary/30">
                  <h2 className="font-semibold">Case Details</h2>
                </div>
                <div className="p-6 space-y-6">
                  {/* Risk Factors */}
                  <div>
                    <div className="text-xs text-muted-foreground mb-3">Risk Factors</div>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                      <div className="p-4 bg-secondary rounded">
                        <div className="text-xs text-muted-foreground">Age</div>
                        <div className="text-lg font-bold">{selected.risk_factors.age}</div>
                      </div>
                      <div className="p-4 bg-secondary rounded">
                        <div className="text-xs text-muted-foreground">Smoker</div>
                        <div className="text-lg font-bold">{selected.risk_factors.smoker ? "Yes" : "No"}</div>
                      </div>
                      <div className="p-4 bg-secondary rounded">
                        <div className="text-xs text-muted-foreground">Coverage</div>
                        <div className="text-lg font-bold">{formatCurrency(selected.risk_factors.coverage_amount)}</div>
                      </div>
                      <div className="p-4 bg-secondary rounded">
                        <div className="text-xs text-muted-foreground">Term</div>
                        <div className="text-lg font-bold">{selected.risk_factors.term_years} years</div>
                      </div>
                    </div>
                  </div>

                  {/* Risk Score */}
                  <div>
                    <div className="text-xs text-muted-foreground mb-3">Risk Assessment</div>
                    <div className="p-4 bg-secondary rounded">
                      <div className="flex justify-between items-center mb-3">
                        <span className="text-sm">Risk Score</span>
                        <span className="text-xl font-bold">{selected.risk_score.score}/100</span>
                      </div>
                      <div className="h-3 bg-background rounded-full overflow-hidden">
                        <div
                          className={`h-full transition-all ${
                            selected.risk_score.score <= 30 ? "bg-emerald-500" :
                            selected.risk_score.score <= 60 ? "bg-yellow-500" : "bg-red-500"
                          }`}
                          style={{ width: `${selected.risk_score.score}%` }}
                        />
                      </div>
                      {selected.risk_score.flags && selected.risk_score.flags.length > 0 && (
                        <div className="flex flex-wrap gap-2 mt-4">
                          {selected.risk_score.flags.map((flag) => (
                            <span key={flag} className="text-xs px-2 py-1 bg-background rounded">
                              {flag}
                            </span>
                          ))}
                        </div>
                      )}
                      {selected.risk_score.recommended && (
                        <div className="mt-4 flex items-center gap-2">
                          <span className="text-sm text-muted-foreground">Recommendation:</span>
                          <span className={`text-xs font-medium px-2 py-1 rounded ${
                            selected.risk_score.recommended === "approved" ? "bg-emerald-500/20 text-emerald-400" :
                            selected.risk_score.recommended === "declined" ? "bg-red-500/20 text-red-400" :
                            "bg-yellow-500/20 text-yellow-400"
                          }`}>
                            {selected.risk_score.recommended.toUpperCase()}
                          </span>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Decision Form */}
                  <div>
                    <div className="text-xs text-muted-foreground mb-3">Make Decision</div>
                    <div className="space-y-4">
                      <div className="flex gap-4">
                        <label className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="radio"
                            name="decision"
                            checked={decision === "approved"}
                            onChange={() => setDecision("approved")}
                            className="w-4 h-4"
                          />
                          <span className="text-sm">Approve</span>
                        </label>
                        <label className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="radio"
                            name="decision"
                            checked={decision === "declined"}
                            onChange={() => setDecision("declined")}
                            className="w-4 h-4"
                          />
                          <span className="text-sm">Decline</span>
                        </label>
                      </div>
                      <div>
                        <label className="text-sm text-muted-foreground block mb-2">Reason (required)</label>
                        <textarea
                          className="w-full h-24 px-3 py-2 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground resize-none"
                          value={reason}
                          onChange={(e) => setReason(e.target.value)}
                          placeholder="Enter your decision reason..."
                        />
                      </div>
                      <button
                        onClick={submitDecision}
                        disabled={deciding || !reason}
                        className="h-10 px-6 text-sm font-medium bg-white text-black rounded hover:bg-white/90 disabled:opacity-50 transition-colors"
                      >
                        {deciding ? "Submitting..." : "Submit Decision"}
                      </button>
                    </div>
                  </div>

                  {/* IDs */}
                  <div className="border-t border-border pt-4 text-xs text-muted-foreground space-y-1">
                    <div>Case ID: {truncateId(selected.id, 12)}</div>
                    <div>Application ID: {truncateId(selected.application_id, 12)}</div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="bg-card border border-border rounded-lg">
                <div className="py-12 text-center text-muted-foreground">
                  Select a case from the queue to review
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
