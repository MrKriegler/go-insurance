import Link from "next/link";

const steps = [
  {
    number: "01",
    title: "Products",
    description: "Browse available term life and whole life products with coverage limits and base rates.",
  },
  {
    number: "02",
    title: "Quotes",
    description: "Get instant pricing based on age, coverage amount, term, and smoking status.",
  },
  {
    number: "03",
    title: "Applications",
    description: "Create an application linking a quote to applicant personal information.",
  },
  {
    number: "04",
    title: "Underwriting",
    description: "Automatic risk scoring with auto-approve, auto-decline, or referral for manual review.",
  },
  {
    number: "05",
    title: "Offers",
    description: "Approved applications receive binding offers valid for 30 days.",
  },
  {
    number: "06",
    title: "Policies",
    description: "Accepted offers become active policies with unique policy numbers.",
  },
];

const features = [
  {
    title: "Automatic Underwriting",
    description: "Risk scoring engine that auto-approves low-risk applicants and refers complex cases for manual review.",
    metric: "<5s",
    metricLabel: "Decision Time",
  },
  {
    title: "Background Processing",
    description: "Async workers handle underwriting decisions and policy issuance without blocking API responses.",
    metric: "100%",
    metricLabel: "Non-blocking",
  },
  {
    title: "RESTful Design",
    description: "Clean JSON API with RFC 7807 error responses, OpenAPI documentation, and predictable resource paths.",
    metric: "20+",
    metricLabel: "Endpoints",
  },
];

export default function Home() {
  return (
    <div>
      {/* Hero Section */}
      <section className="relative min-h-[90vh] flex items-center grid-pattern">
        <div className="absolute inset-0 bg-gradient-to-b from-transparent via-background/50 to-background" />
        <div className="relative max-w-7xl mx-auto px-6 py-32">
          <div className="max-w-4xl">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-border/50 bg-secondary/50 mb-8">
              <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
              <span className="text-xs text-muted-foreground">Portfolio Tech Demo</span>
            </div>
            <h1 className="text-5xl md:text-7xl font-bold tracking-tight mb-6">
              Insurance API
              <br />
              <span className="text-muted-foreground">Infrastructure</span>
            </h1>
            <p className="text-xl text-muted-foreground max-w-2xl mb-12">
              A complete quote-to-policy management system. Built with Go, DynamoDB,
              and production-ready architecture patterns.
            </p>
            <div className="flex flex-wrap gap-4">
              <Link
                href="/demo/journey"
                className="inline-flex h-12 items-center justify-center px-8 text-sm font-medium bg-white text-black rounded hover:bg-white/90 transition-all hover:scale-[1.02]"
              >
                Start Demo
              </Link>
              <Link
                href="/demo/playground"
                className="inline-flex h-12 items-center justify-center px-8 text-sm font-medium border border-border rounded hover:bg-secondary transition-colors"
              >
                API Playground
              </Link>
            </div>
          </div>
        </div>

        {/* Decorative elements */}
        <div className="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-border to-transparent" />
      </section>

      {/* Code Preview Section */}
      <section className="py-24 px-6 border-b border-border/50">
        <div className="max-w-7xl mx-auto">
          <div className="grid lg:grid-cols-2 gap-16 items-center">
            <div>
              <span className="text-xs text-muted-foreground tracking-widest uppercase mb-4 block">
                Simple Integration
              </span>
              <h2 className="text-3xl md:text-4xl font-bold mb-6">
                Three API calls to issue a policy
              </h2>
              <p className="text-muted-foreground mb-8">
                From initial quote to policy issuance, our streamlined API handles
                the complexity of underwriting and risk assessment automatically.
              </p>
              <div className="flex gap-8 text-sm">
                <div>
                  <div className="text-2xl font-bold">6</div>
                  <div className="text-muted-foreground">API Resources</div>
                </div>
                <div>
                  <div className="text-2xl font-bold">{"<"}100ms</div>
                  <div className="text-muted-foreground">Avg Response</div>
                </div>
                <div>
                  <div className="text-2xl font-bold">RFC 7807</div>
                  <div className="text-muted-foreground">Error Format</div>
                </div>
              </div>
            </div>
            <div className="relative">
              <div className="absolute -inset-4 bg-gradient-to-r from-emerald-500/10 via-transparent to-cyan-500/10 rounded-lg blur-2xl" />
              <div className="relative bg-card border border-border rounded-lg overflow-hidden">
                <div className="flex items-center gap-2 px-4 py-3 border-b border-border bg-secondary/50">
                  <span className="w-3 h-3 rounded-full bg-red-500/50" />
                  <span className="w-3 h-3 rounded-full bg-yellow-500/50" />
                  <span className="w-3 h-3 rounded-full bg-green-500/50" />
                  <span className="ml-4 text-xs text-muted-foreground font-mono">create-quote.sh</span>
                </div>
                <pre className="p-6 text-sm overflow-x-auto">
                  <code className="text-muted-foreground">{`curl -X POST http://localhost:8080/api/v1/quotes \\
  -H "Content-Type: application/json" \\
  -d '{
    "product_slug": "term-life-10",
    "coverage_amount": 150000,
    "term_years": 10,
    "age": 35,
    "smoker": false
  }'`}</code>
                </pre>
                <div className="border-t border-border p-4 bg-secondary/30">
                  <div className="text-xs text-muted-foreground mb-2">Response</div>
                  <pre className="text-sm">
                    <code className="text-emerald-400">{`{
  "id": "01JEH1ABC...",
  "monthly_premium": 37.50,
  "status": "priced"
}`}</code>
                  </pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Lifecycle Section */}
      <section className="py-24 px-6 border-b border-border/50">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16">
            <span className="text-xs text-muted-foreground tracking-widest uppercase mb-4 block">
              The Process
            </span>
            <h2 className="text-3xl md:text-4xl font-bold">
              From Quote to Policy
            </h2>
          </div>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-px bg-border">
            {steps.map((step) => (
              <div
                key={step.number}
                className="bg-background p-8 group hover:bg-secondary/50 transition-colors"
              >
                <span className="text-4xl font-bold text-muted-foreground/30 group-hover:text-emerald-500/50 transition-colors">
                  {step.number}
                </span>
                <h3 className="text-lg font-semibold mt-4 mb-2">{step.title}</h3>
                <p className="text-sm text-muted-foreground">{step.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-24 px-6">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16">
            <span className="text-xs text-muted-foreground tracking-widest uppercase mb-4 block">
              Architecture
            </span>
            <h2 className="text-3xl md:text-4xl font-bold">
              Built for Production
            </h2>
          </div>
          <div className="grid md:grid-cols-3 gap-8">
            {features.map((feature) => (
              <div
                key={feature.title}
                className="p-8 border border-border rounded-lg bg-card hover:border-emerald-500/30 transition-colors group"
              >
                <div className="mb-6">
                  <div className="text-4xl font-bold accent-gradient">{feature.metric}</div>
                  <div className="text-xs text-muted-foreground">{feature.metricLabel}</div>
                </div>
                <h3 className="text-lg font-semibold mb-2">{feature.title}</h3>
                <p className="text-sm text-muted-foreground">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-24 px-6 border-t border-border/50">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl md:text-4xl font-bold mb-6">
            Ready to explore?
          </h2>
          <p className="text-muted-foreground mb-8 max-w-xl mx-auto">
            Walk through the complete insurance lifecycle with our interactive demo,
            or dive into the API with our playground.
          </p>
          <div className="flex flex-wrap justify-center gap-4">
            <Link
              href="/demo/journey"
              className="inline-flex h-12 items-center justify-center px-8 text-sm font-medium bg-white text-black rounded hover:bg-white/90 transition-all"
            >
              Start Journey Demo
            </Link>
            <a
              href="http://localhost:8080/swagger/"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex h-12 items-center justify-center px-8 text-sm font-medium border border-border rounded hover:bg-secondary transition-colors"
            >
              View Swagger Docs
            </a>
          </div>
        </div>
      </section>
    </div>
  );
}
