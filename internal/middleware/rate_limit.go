package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RateLimiter struct {
	requests   map[string][]time.Time
	mu         sync.Mutex
	limit      int
	window     time.Duration
	trustProxy bool
}

// NewRateLimiter creates an in-memory rate limiter. trustProxy must be true
// when the server runs behind Cloudflare (or another trusted proxy) so the
// real client IP from CF-Connecting-IP is used instead of the proxy's IP.
func NewRateLimiter(limit int, window time.Duration, trustProxy bool) *RateLimiter {
	rl := &RateLimiter{
		requests:   make(map[string][]time.Time),
		limit:      limit,
		window:     window,
		trustProxy: trustProxy,
	}
	go rl.cleanup()
	return rl
}

func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ip := resolveClientIP(req, r.trustProxy)
		if !r.allow(ip) {
			w.Header().Set("Retry-After", strconv.Itoa(int(r.window.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error": map[string]string{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests",
				},
			})
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (r *RateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.window)

	var filtered []time.Time
	for _, t := range r.requests[key] {
		if t.After(windowStart) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= r.limit {
		return false
	}

	filtered = append(filtered, now)
	r.requests[key] = filtered
	return true
}

func (r *RateLimiter) cleanup() {
	for {
		time.Sleep(r.window)
		r.mu.Lock()
		r.requests = make(map[string][]time.Time)
		r.mu.Unlock()
	}
}
