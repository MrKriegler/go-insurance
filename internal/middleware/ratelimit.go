package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MrKriegler/go-insurance/pkg/problem"
)

// RateLimiter provides simple in-memory rate limiting for demo purposes.
// In production, use Redis-based distributed rate limiting.
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int           // max requests
	window   time.Duration // time window
	stopCh   chan struct{}
}

// NewRateLimiter creates a rate limiter with the given limit per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		stopCh:   make(chan struct{}),
	}
	// Cleanup old entries every minute
	go rl.cleanup()
	return rl
}

// Stop gracefully stops the rate limiter's cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// StartWithContext starts cleanup with context cancellation support.
func (rl *RateLimiter) StartWithContext(ctx context.Context) {
	go func() {
		<-ctx.Done()
		rl.Stop()
	}()
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, times := range rl.requests {
				var valid []time.Time
				for _, t := range times {
					if now.Sub(t) < rl.window {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(rl.requests, ip)
				} else {
					rl.requests[ip] = valid
				}
			}
			rl.mu.Unlock()
		}
	}
}

func (rl *RateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Filter to only requests within window
	var valid []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[ip] = valid
		return false
	}

	rl.requests[ip] = append(valid, now)
	return true
}

// Middleware returns the rate limiting middleware handler.
// NOTE: This should be used AFTER chi's RealIP middleware which safely
// sets RemoteAddr from X-Forwarded-For when behind trusted proxies.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use RemoteAddr which is set by chi's RealIP middleware when behind proxy.
		// Do NOT trust X-Real-IP or X-Forwarded-For directly as clients can spoof them.
		ip := r.RemoteAddr

		// Strip port if present (RemoteAddr is usually "ip:port")
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			// Check if it's IPv6 (contains multiple colons)
			if strings.Count(ip, ":") > 1 {
				// IPv6: look for ]:port pattern
				if bracketIdx := strings.LastIndex(ip, "]:"); bracketIdx != -1 {
					ip = ip[1:bracketIdx] // Remove [ and ]:port
				}
			} else {
				ip = ip[:idx]
			}
		}

		if !rl.isAllowed(ip) {
			w.Header().Set("Retry-After", "60")
			problem.Write(w, http.StatusTooManyRequests, "Rate Limit Exceeded",
				"Too many requests. Please try again later.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
