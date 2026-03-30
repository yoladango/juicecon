package weather

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCelsiusToFahrenheit(t *testing.T) {
	tests := []struct {
		name    string
		celsius float64
		wantF   float64
	}{
		{
			name:    "freezing point",
			celsius: 0,
			wantF:   32,
		},
		{
			name:    "boiling point",
			celsius: 100,
			wantF:   212,
		},
		{
			name:    "negative 40 crossover",
			celsius: -40,
			wantF:   -40,
		},
		{
			name:    "body temperature",
			celsius: 37,
			wantF:   98.6,
		},
		{
			name:    "sub-zero celsius",
			celsius: -20,
			wantF:   -4,
		},
		{
			name:    "mild temperature",
			celsius: 20,
			wantF:   68,
		},
	}

	const tolerance = 0.01

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := celsiusToFahrenheit(tt.celsius)
			if math.Abs(got-tt.wantF) > tolerance {
				t.Errorf("celsiusToFahrenheit(%g) = %g, want %g", tt.celsius, got, tt.wantF)
			}
		})
	}
}

// urlRewriteDoer wraps an http.Client and rewrites all request URLs to point
// at the test server, preserving the original path and query.
type urlRewriteDoer struct {
	target     string // base URL of the httptest.Server (e.g. http://127.0.0.1:PORT)
	underlying *http.Client
}

func (d *urlRewriteDoer) Do(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point at the test server while keeping the path.
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(d.target, "http://")
	return d.underlying.Do(req)
}

// newTestClient creates a Client whose HTTP requests are redirected to the
// given httptest.Server.
func newTestClient(ts *httptest.Server) *Client {
	return &Client{
		httpClient: &urlRewriteDoer{
			target:     ts.URL,
			underlying: ts.Client(),
		},
		cache: newObservationCache(defaultCacheTTL),
	}
}

// Helper to build a valid points JSON response. The stationsURL should point
// to the path that the test server will serve for the stations endpoint.
func pointsJSON(stationsURL, city, state string) string {
	return fmt.Sprintf(`{
		"properties": {
			"observationStations": %q,
			"relativeLocation": {
				"properties": {
					"city": %q,
					"state": %q
				}
			}
		}
	}`, stationsURL, city, state)
}

func stationsJSON(stationIDs ...string) string {
	if len(stationIDs) == 0 {
		return `{"features": []}`
	}
	var features []string
	for _, id := range stationIDs {
		features = append(features, fmt.Sprintf(`{"properties": {"stationIdentifier": %q}}`, id))
	}
	return fmt.Sprintf(`{"features": [%s]}`, strings.Join(features, ","))
}

func observationJSON(dewpointC, tempC *float64, timestamp string) string {
	dpVal := "null"
	if dewpointC != nil {
		dpVal = fmt.Sprintf("%g", *dewpointC)
	}
	tVal := "null"
	if tempC != nil {
		tVal = fmt.Sprintf("%g", *tempC)
	}
	return fmt.Sprintf(`{
		"properties": {
			"dewpoint": {"value": %s, "unitCode": "wmoUnit:degC"},
			"temperature": {"value": %s, "unitCode": "wmoUnit:degC"},
			"timestamp": %q
		}
	}`, dpVal, tVal, timestamp)
}

func floatPtr(v float64) *float64 {
	return &v
}

func TestGetObservation_HappyPath(t *testing.T) {
	dewC := 20.0  // 68°F
	tempC := 25.0 // 77°F
	ts := "2026-03-30T12:00:00Z"

	mux := http.NewServeMux()
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		// The stations URL must also go through the test server, so we use
		// the test server's own origin. We'll discover it from the request host.
		stationsURL := fmt.Sprintf("http://%s/stations-for-test", r.Host)
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, pointsJSON(stationsURL, "Chicago", "IL"))
	})
	mux.HandleFunc("/stations-for-test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, stationsJSON("KORD"))
	})
	mux.HandleFunc("/stations/KORD/observations/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, observationJSON(&dewC, &tempC, ts))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server)
	obs, err := client.GetObservation(context.Background(), 41.8781, -87.6298)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obs.Station != "KORD" {
		t.Errorf("Station = %q, want %q", obs.Station, "KORD")
	}
	if obs.City != "Chicago" {
		t.Errorf("City = %q, want %q", obs.City, "Chicago")
	}
	if obs.State != "IL" {
		t.Errorf("State = %q, want %q", obs.State, "IL")
	}
	if math.Abs(obs.DewpointC-dewC) > 0.01 {
		t.Errorf("DewpointC = %g, want %g", obs.DewpointC, dewC)
	}
	wantDewF := celsiusToFahrenheit(dewC)
	if math.Abs(obs.DewpointF-wantDewF) > 0.01 {
		t.Errorf("DewpointF = %g, want %g", obs.DewpointF, wantDewF)
	}
	wantTempF := celsiusToFahrenheit(tempC)
	if obs.TemperatureF == nil {
		t.Fatal("TemperatureF should not be nil")
	}
	if math.Abs(*obs.TemperatureF-wantTempF) > 0.01 {
		t.Errorf("TemperatureF = %g, want %g", *obs.TemperatureF, wantTempF)
	}
	if obs.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestGetObservation_PointsAPI500(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetObservation(context.Background(), 41.8781, -87.6298)
	if err == nil {
		t.Fatal("expected error when points API returns 500, got nil")
	}
	if !strings.Contains(err.Error(), "points lookup failed") {
		t.Errorf("error should mention points lookup, got: %v", err)
	}
}

func TestGetObservation_EmptyStationsList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		stationsURL := fmt.Sprintf("http://%s/stations-empty", r.Host)
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, pointsJSON(stationsURL, "Nowhere", "XX"))
	})
	mux.HandleFunc("/stations-empty", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, stationsJSON()) // empty features
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetObservation(context.Background(), 41.8781, -87.6298)
	if err == nil {
		t.Fatal("expected error for empty stations list, got nil")
	}
	if !strings.Contains(err.Error(), "no observation stations found") {
		t.Errorf("error should mention no stations, got: %v", err)
	}
}

func TestGetObservation_NilDewpoint(t *testing.T) {
	tempC := 25.0
	mux := http.NewServeMux()
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		stationsURL := fmt.Sprintf("http://%s/stations-dp", r.Host)
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, pointsJSON(stationsURL, "Chicago", "IL"))
	})
	mux.HandleFunc("/stations-dp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, stationsJSON("KORD"))
	})
	mux.HandleFunc("/stations/KORD/observations/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, observationJSON(nil, &tempC, "2026-03-30T12:00:00Z"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetObservation(context.Background(), 41.8781, -87.6298)
	if err == nil {
		t.Fatal("expected error for nil dewpoint, got nil")
	}
	if !strings.Contains(err.Error(), "dewpoint data not available") {
		t.Errorf("error should mention dewpoint not available, got: %v", err)
	}
}

func TestGetObservation_NilTemperatureReturnsNil(t *testing.T) {
	dewC := 15.0
	mux := http.NewServeMux()
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		stationsURL := fmt.Sprintf("http://%s/stations-nt", r.Host)
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, pointsJSON(stationsURL, "Chicago", "IL"))
	})
	mux.HandleFunc("/stations-nt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, stationsJSON("KORD"))
	})
	mux.HandleFunc("/stations/KORD/observations/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		// nil temperature
		fmt.Fprint(w, observationJSON(&dewC, nil, "2026-03-30T12:00:00Z"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server)
	obs, err := client.GetObservation(context.Background(), 41.8781, -87.6298)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.TemperatureF != nil {
		t.Errorf("TemperatureF = %g, want nil for unavailable temperature", *obs.TemperatureF)
	}
	// Dewpoint should still be valid
	wantDewF := celsiusToFahrenheit(dewC)
	if math.Abs(obs.DewpointF-wantDewF) > 0.01 {
		t.Errorf("DewpointF = %g, want %g", obs.DewpointF, wantDewF)
	}
}

// errorDoer is an HTTPDoer that always returns an error, simulating
// a network failure or unreachable host.
type errorDoer struct {
	err error
}

func (d *errorDoer) Do(req *http.Request) (*http.Response, error) {
	return nil, d.err
}

func TestGetObservation_NetworkError(t *testing.T) {
	client := &Client{
		httpClient: &errorDoer{err: fmt.Errorf("connection refused")},
		cache:      newObservationCache(defaultCacheTTL),
	}

	_, err := client.GetObservation(context.Background(), 41.8781, -87.6298)
	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
	if !strings.Contains(err.Error(), "points lookup failed") {
		t.Errorf("error should mention points lookup failed, got: %v", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should contain underlying cause, got: %v", err)
	}
}

func TestGetObservation_MalformedJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/points/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/geo+json")
		fmt.Fprint(w, `{not valid json!!!}`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetObservation(context.Background(), 41.8781, -87.6298)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode points response") {
		t.Errorf("error should mention decode failure, got: %v", err)
	}
}
