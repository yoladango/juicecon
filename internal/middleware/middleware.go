package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SecurityHeaders wraps a handler and sets security-related response headers
// on every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// visitor tracks request counts for a single IP address.
type visitor struct {
	count    int
	windowStart time.Time
}

// RateLimiter provides per-IP rate limiting for HTTP handlers.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
	done     chan struct{}
}

// NewRateLimiter creates a rate limiter that allows limit requests per window
// per IP. It starts a background goroutine that cleans up stale entries every
// cleanupInterval. Call Stop() to terminate the cleanup goroutine.
func NewRateLimiter(limit int, window, cleanupInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
		done:     make(chan struct{}),
	}

	go rl.cleanupLoop(cleanupInterval)
	return rl
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.done)
}

// Allow checks whether the given IP is within its rate limit. It returns true
// if the request should be allowed, false if it should be rejected.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists || now.Sub(v.windowStart) >= rl.window {
		rl.visitors[ip] = &visitor{count: 1, windowStart: now}
		return true
	}

	v.count++
	return v.count <= rl.limit
}

// Wrap returns middleware that applies rate limiting to the given handler.
// Only requests whose path starts with pathPrefix are rate-limited; all
// others pass through directly.
func (rl *RateLimiter) Wrap(pathPrefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, pathPrefix) {
			ip := extractIP(r)
			if !rl.Allow(ip) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// cleanupLoop removes stale visitor entries periodically.
func (rl *RateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.done:
			return
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, v := range rl.visitors {
		if now.Sub(v.windowStart) >= rl.window {
			delete(rl.visitors, ip)
		}
	}
	log.Printf("Rate limiter cleanup: %d active visitors", len(rl.visitors))
}

// extractIP pulls the client IP from X-Forwarded-For, X-Real-IP, or
// falls back to RemoteAddr.
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For first (common behind reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.IndexByte(xff, ','); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Strip port from RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
