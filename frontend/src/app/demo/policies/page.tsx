"use client";

import { useState, useEffect } from "react";
import { api, ApiError } from "@/lib/api";
import { formatCurrency, formatDate, truncateId } from "@/lib/utils";
import type { Policy } from "@/lib/types";

export default function PoliciesPage() {
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [total, setTotal] = useState(0);
  const [selected, setSelected] = useState<Policy | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPolicies = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.policies.list({ limit: 50 });
      setPolicies(data.items || []);
      setTotal(data.total || 0);
    } catch (e) {
      const err = e as ApiError;
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPolicies();
  }, []);

  const statusColor = (status: Policy["status"]) => {
    switch (status) {
      case "active": return "text-emerald-400";
      case "lapsed":
      case "cancelled": return "text-red-400";
      default: return "text-muted-foreground";
    }
  };

  return (
    <div className="min-h-screen">
      {/* Header */}
      <div className="border-b border-border/50 bg-card">
        <div className="max-w-7xl mx-auto px-6 py-8">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold mb-2">Policies</h1>
              <p className="text-muted-foreground">View issued policies</p>
            </div>
            <button
              onClick={loadPolicies}
              disabled={loading}
              className="h-10 px-6 text-sm font-medium border border-border rounded hover:bg-secondary disabled:opacity-50 transition-colors"
            >
              {loading ? "Loading..." : "Refresh"}
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
          {/* Policy Table */}
          <div className="lg:col-span-2">
            <div className="bg-card border border-border rounded-lg overflow-hidden">
              <div className="px-6 py-4 border-b border-border bg-secondary/30">
                <h2 className="font-semibold">All Policies ({total})</h2>
              </div>
              {loading ? (
                <div className="py-12 text-center">
                  <div className="inline-block w-6 h-6 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  <p className="text-muted-foreground mt-4">Loading policies...</p>
                </div>
              ) : policies.length === 0 ? (
                <div className="py-12 text-center text-muted-foreground">
                  No policies found. Complete a journey demo to issue a policy.
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border bg-secondary/30">
                        <th className="text-left px-6 py-3 text-muted-foreground font-medium">Policy #</th>
                        <th className="text-left px-6 py-3 text-muted-foreground font-medium">Insured</th>
                        <th className="text-right px-6 py-3 text-muted-foreground font-medium">Coverage</th>
                        <th className="text-right px-6 py-3 text-muted-foreground font-medium">Premium</th>
                        <th className="text-center px-6 py-3 text-muted-foreground font-medium">Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      {policies.map((p) => (
                        <tr
                          key={p.id}
                          onClick={() => setSelected(p)}
                          className={`border-b border-border cursor-pointer transition-colors ${
                            selected?.id === p.id ? "bg-secondary/50" : "hover:bg-secondary/30"
                          }`}
                        >
                          <td className="px-6 py-4 font-mono">{p.number}</td>
                          <td className="px-6 py-4">{p.insured.first_name} {p.insured.last_name}</td>
                          <td className="px-6 py-4 text-right">{formatCurrency(p.coverage_amount)}</td>
                          <td className="px-6 py-4 text-right">{formatCurrency(p.monthly_premium)}/mo</td>
                          <td className="px-6 py-4 text-center">
                            <span className={`text-xs font-medium ${statusColor(p.status)}`}>
                              {p.status.toUpperCase()}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>

          {/* Policy Details */}
          <div className="lg:col-span-1">
            {selected ? (
              <div className="bg-card border border-border rounded-lg overflow-hidden">
                <div className="px-6 py-4 border-b border-border bg-secondary/30">
                  <h2 className="font-semibold font-mono">{selected.number}</h2>
                </div>
                <div className="p-6 space-y-6">
                  <div className="text-center pb-4 border-b border-border">
                    <span className={`text-lg font-semibold ${statusColor(selected.status)}`}>
                      {selected.status.toUpperCase()}
                    </span>
                  </div>

                  <div>
                    <div className="text-xs text-muted-foreground mb-1">Coverage</div>
                    <div className="text-2xl font-bold">{formatCurrency(selected.coverage_amount)}</div>
                    <div className="text-sm text-muted-foreground">{selected.term_years} year term</div>
                  </div>

                  <div>
                    <div className="text-xs text-muted-foreground mb-1">Premium</div>
                    <div className="text-xl font-bold">{formatCurrency(selected.monthly_premium)}/month</div>
                  </div>

                  <div className="border-t border-border pt-4">
                    <div className="text-xs text-muted-foreground mb-3">Insured</div>
                    <div className="font-semibold">{selected.insured.first_name} {selected.insured.last_name}</div>
                    <div className="text-sm text-muted-foreground">{selected.insured.email}</div>
                    <div className="text-sm text-muted-foreground">
                      DOB: {selected.insured.date_of_birth} | {selected.insured.state}
                    </div>
                  </div>

                  <div className="border-t border-border pt-4">
                    <div className="text-xs text-muted-foreground mb-3">Dates</div>
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <div className="text-muted-foreground">Issued</div>
                        <div>{formatDate(selected.issued_at)}</div>
                      </div>
                      <div>
                        <div className="text-muted-foreground">Effective</div>
                        <div>{formatDate(selected.effective_date)}</div>
                      </div>
                      <div className="col-span-2">
                        <div className="text-muted-foreground">Expires</div>
                        <div>{formatDate(selected.expiry_date)}</div>
                      </div>
                    </div>
                  </div>

                  <div className="border-t border-border pt-4 text-xs text-muted-foreground space-y-1">
                    <div>Policy ID: {truncateId(selected.id, 12)}</div>
                    <div>Application: {truncateId(selected.application_id, 12)}</div>
                    <div>Offer: {truncateId(selected.offer_id, 12)}</div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="bg-card border border-border rounded-lg">
                <div className="py-12 text-center text-muted-foreground">
                  Select a policy to view details
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
