"use client";

import { useState } from "react";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";

interface Scenario {
  group: string;
  name: string;
  method: "GET" | "POST" | "PATCH";
  path: string;
  pathParams: { name: string; default: string }[];
  body: object | null;
}

const scenarios: Scenario[] = [
  { group: "Products", name: "List all products", method: "GET", path: "/products", pathParams: [], body: null },
  { group: "Products", name: "Get product by slug", method: "GET", path: "/products/{product_slug}", pathParams: [{ name: "product_slug", default: "term-life-10" }], body: null },
  { group: "Quotes", name: "Create quote", method: "POST", path: "/quotes", pathParams: [], body: { product_slug: "term-life-10", coverage_amount: 150000, term_years: 10, age: 35, smoker: false } },
  { group: "Quotes", name: "Get quote", method: "GET", path: "/quotes/{quote_id}", pathParams: [{ name: "quote_id", default: "" }], body: null },
  { group: "Applications", name: "Create application", method: "POST", path: "/applications", pathParams: [], body: { quote_id: "", applicant: { first_name: "John", last_name: "Doe", email: "john@example.com", date_of_birth: "1989-06-15", age: 35, smoker: false, state: "CA" } } },
  { group: "Applications", name: "Get application", method: "GET", path: "/applications/{application_id}", pathParams: [{ name: "application_id", default: "" }], body: null },
  { group: "Applications", name: "Submit application", method: "POST", path: "/applications/{application_id}:submit", pathParams: [{ name: "application_id", default: "" }], body: null },
  { group: "Underwriting", name: "List referred cases", method: "GET", path: "/underwriting/cases", pathParams: [], body: null },
  { group: "Underwriting", name: "Get case details", method: "GET", path: "/underwriting/cases/{case_id}", pathParams: [{ name: "case_id", default: "" }], body: null },
  { group: "Underwriting", name: "Decide case", method: "POST", path: "/underwriting/cases/{case_id}:decide", pathParams: [{ name: "case_id", default: "" }], body: { decision: "approved", reason: "Manual review complete" } },
  { group: "Offers", name: "Generate offer", method: "POST", path: "/applications/{application_id}/offers", pathParams: [{ name: "application_id", default: "" }], body: null },
  { group: "Offers", name: "Get offer", method: "GET", path: "/offers/{offer_id}", pathParams: [{ name: "offer_id", default: "" }], body: null },
  { group: "Offers", name: "Accept offer", method: "POST", path: "/offers/{offer_id}:accept", pathParams: [{ name: "offer_id", default: "" }], body: null },
  { group: "Offers", name: "Decline offer", method: "POST", path: "/offers/{offer_id}:decline", pathParams: [{ name: "offer_id", default: "" }], body: null },
  { group: "Policies", name: "List policies", method: "GET", path: "/policies", pathParams: [], body: null },
  { group: "Policies", name: "Get policy by number", method: "GET", path: "/policies/{policy_number}", pathParams: [{ name: "policy_number", default: "POL-2025-000001" }], body: null },
];

export default function PlaygroundPage() {
  const [selected, setSelected] = useState<Scenario>(scenarios[0]);
  const [pathParams, setPathParams] = useState<Record<string, string>>({});
  const [requestBody, setRequestBody] = useState<string>("");
  const [response, setResponse] = useState<{ status: number; data: object; time: number } | null>(null);
  const [loading, setLoading] = useState(false);

  const selectScenario = (scenario: Scenario) => {
    setSelected(scenario);
    const params: Record<string, string> = {};
    scenario.pathParams.forEach((p) => {
      params[p.name] = p.default;
    });
    setPathParams(params);
    setRequestBody(scenario.body ? JSON.stringify(scenario.body, null, 2) : "");
    setResponse(null);
  };

  const buildPath = () => {
    let path = selected.path;
    selected.pathParams.forEach((p) => {
      path = path.replace(`{${p.name}}`, pathParams[p.name] || `{${p.name}}`);
    });
    return path;
  };

  const sendRequest = async () => {
    setLoading(true);
    const start = Date.now();
    try {
      const url = `${API_BASE_URL}${buildPath()}`;
      const options: RequestInit = {
        method: selected.method,
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": process.env.NEXT_PUBLIC_API_KEY || "demo-api-key-12345",
        },
      };
      if (requestBody && selected.method !== "GET") {
        options.body = requestBody;
      }
      const res = await fetch(url, options);
      const data = await res.json();
      setResponse({ status: res.status, data, time: Date.now() - start });
    } catch (e) {
      setResponse({ status: 0, data: { error: String(e) }, time: Date.now() - start });
    } finally {
      setLoading(false);
    }
  };

  const groups = [...new Set(scenarios.map((s) => s.group))];

  const methodColor = (method: string) => {
    switch (method) {
      case "GET": return "bg-emerald-500/20 text-emerald-400";
      case "POST": return "bg-blue-500/20 text-blue-400";
      case "PATCH": return "bg-yellow-500/20 text-yellow-400";
      default: return "bg-secondary text-muted-foreground";
    }
  };

  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <div className="border-b border-border/50 bg-card">
        <div className="max-w-7xl mx-auto px-6 py-8">
          <h1 className="text-3xl font-bold mb-2">API Playground</h1>
          <p className="text-muted-foreground">Explore and test API endpoints interactively</p>
        </div>
      </div>

      <div className="flex-1 flex">
        {/* Sidebar */}
        <div className="w-64 border-r border-border bg-card overflow-y-auto">
          <div className="p-4">
            <div className="text-xs text-muted-foreground uppercase tracking-wider mb-4">Endpoints</div>
            {groups.map((group) => (
              <div key={group} className="mb-6">
                <div className="text-sm font-medium text-muted-foreground mb-2">{group}</div>
                <div className="space-y-1">
                  {scenarios
                    .filter((s) => s.group === group)
                    .map((s) => (
                      <button
                        key={s.name}
                        onClick={() => selectScenario(s)}
                        className={`w-full text-left px-3 py-2 text-sm rounded transition-colors ${
                          selected.name === s.name
                            ? "bg-white text-black"
                            : "text-muted-foreground hover:bg-secondary hover:text-foreground"
                        }`}
                      >
                        {s.name}
                      </button>
                    ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Main Content */}
        <div className="flex-1 flex">
          {/* Request Builder */}
          <div className="flex-1 border-r border-border p-6 overflow-y-auto">
            <div className="space-y-6">
              <div>
                <div className="text-xs text-muted-foreground uppercase tracking-wider mb-3">Request</div>
                <div className="flex items-center gap-3 p-4 bg-secondary rounded">
                  <span className={`text-xs font-medium px-2 py-1 rounded ${methodColor(selected.method)}`}>
                    {selected.method}
                  </span>
                  <code className="text-sm flex-1 text-muted-foreground break-all">{API_BASE_URL}{buildPath()}</code>
                </div>
              </div>

              {selected.pathParams.length > 0 && (
                <div>
                  <div className="text-xs text-muted-foreground uppercase tracking-wider mb-3">Path Parameters</div>
                  <div className="space-y-3">
                    {selected.pathParams.map((p) => (
                      <div key={p.name}>
                        <label className="text-sm text-muted-foreground block mb-2">{p.name}</label>
                        <input
                          value={pathParams[p.name] || ""}
                          onChange={(e) => setPathParams({ ...pathParams, [p.name]: e.target.value })}
                          placeholder={`Enter ${p.name}`}
                          className="w-full h-10 px-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground font-mono text-sm"
                        />
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {selected.body && (
                <div>
                  <div className="text-xs text-muted-foreground uppercase tracking-wider mb-3">Request Body</div>
                  <textarea
                    className="w-full h-64 px-4 py-3 bg-secondary border border-border rounded text-foreground focus:outline-none focus:border-muted-foreground font-mono text-sm resize-none"
                    value={requestBody}
                    onChange={(e) => setRequestBody(e.target.value)}
                  />
                </div>
              )}

              <button
                onClick={sendRequest}
                disabled={loading}
                className="w-full h-10 text-sm font-medium bg-white text-black rounded hover:bg-white/90 disabled:opacity-50 transition-colors"
              >
                {loading ? "Sending..." : "Send Request"}
              </button>
            </div>
          </div>

          {/* Response Viewer */}
          <div className="flex-1 p-6 overflow-y-auto bg-secondary/30">
            <div className="text-xs text-muted-foreground uppercase tracking-wider mb-3">Response</div>
            {response ? (
              <div className="space-y-4">
                <div className="flex items-center gap-3">
                  <span className={`text-xs font-medium px-2 py-1 rounded ${
                    response.status < 400 ? "bg-emerald-500/20 text-emerald-400" : "bg-red-500/20 text-red-400"
                  }`}>
                    {response.status}
                  </span>
                  <span className="text-sm text-muted-foreground">{response.time}ms</span>
                </div>
                <pre className="p-4 bg-card border border-border rounded text-sm overflow-x-auto font-mono">
                  {JSON.stringify(response.data, null, 2)}
                </pre>
              </div>
            ) : (
              <div className="text-muted-foreground text-sm">
                Send a request to see the response
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
