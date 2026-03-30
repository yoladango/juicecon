package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeaders(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expected := map[string]string{
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
	}

	for header, want := range expected {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute, time.Hour)
	defer rl.Stop()

	ip := "192.168.1.1"

	for i := 0; i < 3; i++ {
		if !rl.Allow(ip) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if rl.Allow(ip) {
		t.Error("request 4 should be rejected")
	}

	// A different IP should still be allowed
	if !rl.Allow("10.0.0.1") {
		t.Error("different IP should be allowed")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter(2, 50*time.Millisecond, time.Hour)
	defer rl.Stop()

	ip := "192.168.1.1"

	rl.Allow(ip)
	rl.Allow(ip)
	if rl.Allow(ip) {
		t.Error("should be rate limited")
	}

	// Wait for the window to expire
	time.Sleep(60 * time.Millisecond)

	if !rl.Allow(ip) {
		t.Error("should be allowed after window reset")
	}
}

func TestRateLimiter_Wrap_OnlyAPIRoutes(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute, time.Hour)
	defer rl.Stop()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rl.Wrap("/api/", inner)

	// First API request: allowed
	req := httptest.NewRequest(http.MethodGet, "/api/dewcon?zip=43215", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("first API request: got %d, want 200", rec.Code)
	}

	// Second API request: rate limited
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("second API request: got %d, want 429", rec.Code)
	}

	// Non-API request from same IP: should pass through
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("static request: got %d, want 200", rec.Code)
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		want       string
	}{
		{
			name:       "plain remote addr",
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For single",
			remoteAddr: "127.0.0.1:1234",
			xff:        "10.0.0.1",
			want:       "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For chain",
			remoteAddr: "127.0.0.1:1234",
			xff:        "10.0.0.1, 10.0.0.2, 10.0.0.3",
			want:       "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "127.0.0.1:1234",
			xri:        "172.16.0.1",
			want:       "172.16.0.1",
		},
		{
			name:       "XFF takes precedence over XRI",
			remoteAddr: "127.0.0.1:1234",
			xff:        "10.0.0.1",
			xri:        "172.16.0.1",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			got := extractIP(req)
			if got != tt.want {
				t.Errorf("extractIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(5, 50*time.Millisecond, time.Hour)
	defer rl.Stop()

	rl.Allow("1.1.1.1")
	rl.Allow("2.2.2.2")

	// Both should be present
	rl.mu.Lock()
	if len(rl.visitors) != 2 {
		t.Errorf("expected 2 visitors, got %d", len(rl.visitors))
	}
	rl.mu.Unlock()

	// Wait for window to expire, then trigger cleanup
	time.Sleep(60 * time.Millisecond)
	rl.cleanup()

	rl.mu.Lock()
	if len(rl.visitors) != 0 {
		t.Errorf("expected 0 visitors after cleanup, got %d", len(rl.visitors))
	}
	rl.mu.Unlock()
}
