import type { Metadata } from "next";
import { Inter } from "next/font/google";
import Link from "next/link";
import "./globals.css";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });

export const metadata: Metadata = {
  title: "Go Insurance API",
  description: "Life Insurance Quote and Policy Management API Demo",
};

function Navigation() {
  return (
    <nav className="fixed top-0 left-0 right-0 z-50 border-b border-border/50 bg-background/80 backdrop-blur-md">
      <div className="max-w-7xl mx-auto px-6">
        <div className="flex justify-between h-14">
          <div className="flex items-center gap-12">
            <Link href="/" className="flex items-center gap-2">
              <div className="w-6 h-6 bg-white rounded-sm flex items-center justify-center">
                <span className="text-black text-xs font-bold">GI</span>
              </div>
              <span className="text-sm font-medium tracking-wide">GO INSURANCE</span>
            </Link>
            <div className="hidden md:flex items-center gap-8">
              <Link
                href="/demo/journey"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Journey
              </Link>
              <Link
                href="/demo/playground"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Playground
              </Link>
              <Link
                href="/demo/underwriting"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Underwriting
              </Link>
              <Link
                href="/demo/policies"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Policies
              </Link>
            </div>
          </div>
          <div className="flex items-center gap-6">
            <a
              href="http://localhost:8080/swagger/"
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              API Docs
            </a>
            <Link
              href="/demo/journey"
              className="hidden sm:flex h-8 items-center px-4 text-sm font-medium bg-white text-black rounded hover:bg-white/90 transition-colors"
            >
              Start Demo
            </Link>
          </div>
        </div>
      </div>
    </nav>
  );
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={inter.variable}>
      <body className="font-sans">
        <div className="min-h-screen flex flex-col">
          <Navigation />
          <main className="flex-1 pt-14">{children}</main>
          <footer className="border-t border-border/50 py-8 mt-auto">
            <div className="max-w-7xl mx-auto px-6">
              <div className="flex flex-col md:flex-row justify-between items-center gap-4">
                <div className="flex items-center gap-2">
                  <div className="w-5 h-5 bg-white rounded-sm flex items-center justify-center">
                    <span className="text-black text-[10px] font-bold">GI</span>
                  </div>
                  <span className="text-xs text-muted-foreground">
                    Go Insurance API Demo
                  </span>
                </div>
                <div className="flex items-center gap-6 text-xs text-muted-foreground">
                  <span>Built with Go + DynamoDB</span>
                  <span className="hidden sm:inline">|</span>
                  <a
                    href="https://github.com"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-foreground transition-colors"
                  >
                    GitHub
                  </a>
                </div>
              </div>
            </div>
          </footer>
        </div>
      </body>
    </html>
  );
}
