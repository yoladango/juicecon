package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"juicecon-golang/internal/weather"
)

// mockWeatherClient implements WeatherClient for testing.
type mockWeatherClient struct {
	observation *weather.Observation
	err         error
}

func (m *mockWeatherClient) GetObservation(_ context.Context, _, _ float64) (*weather.Observation, error) {
	return m.observation, m.err
}

func TestTestModeDewpoint71(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodGet, "/api/juicecon?dewpoint=71", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ActiveSystem != "juicecon" {
		t.Errorf("expected activeSystem juicecon, got %q", resp.ActiveSystem)
	}
	if resp.Level == nil || *resp.Level != 3 {
		t.Errorf("expected level 3 (JC3), got %v", resp.Level)
	}
	if resp.LevelDisplay != "JUICECON 3" {
		t.Errorf("expected levelDisplay JUICECON 3, got %q", resp.LevelDisplay)
	}
	if resp.Dewpoint != 71.0 {
		t.Errorf("expected dewpoint 71.0, got %f", resp.Dewpoint)
	}
	if resp.AllClear {
		t.Error("expected allClear to be false")
	}
	if resp.Location.City != "Test Mode" {
		t.Errorf("expected city Test Mode, got %q", resp.Location.City)
	}
}

func TestTestModeExtremeCold(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodGet, "/api/juicecon?dewpoint=-5", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ActiveSystem != "ccf" {
		t.Errorf("expected activeSystem ccf, got %q", resp.ActiveSystem)
	}
	if resp.Level == nil || *resp.Level != 1 {
		t.Errorf("expected level 1 (CCF1), got %v", resp.Level)
	}
	if resp.LevelDisplay != "CCF 1" {
		t.Errorf("expected levelDisplay CCF 1, got %q", resp.LevelDisplay)
	}
	if resp.AllClear {
		t.Error("expected allClear to be false")
	}
}

func TestTestModeComfortZone(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodGet, "/api/juicecon?dewpoint=45", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ActiveSystem != "none" {
		t.Errorf("expected activeSystem none, got %q", resp.ActiveSystem)
	}
	if resp.Level != nil {
		t.Errorf("expected nil level, got %d", *resp.Level)
	}
	if resp.LevelDisplay != "ALL CLEAR" {
		t.Errorf("expected levelDisplay ALL CLEAR, got %q", resp.LevelDisplay)
	}
	if !resp.AllClear {
		t.Error("expected allClear to be true")
	}
}

func TestInvalidMethod(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodPost, "/api/juicecon", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("expected error code METHOD_NOT_ALLOWED, got %q", errResp.Code)
	}
}

func TestMissingParameters(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodGet, "/api/juicecon", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != "INVALID_PARAMS" {
		t.Errorf("expected error code INVALID_PARAMS, got %q", errResp.Code)
	}
}

func TestInvalidZIP(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodGet, "/api/juicecon?zip=00000", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != "INVALID_PARAMS" {
		t.Errorf("expected error code INVALID_PARAMS, got %q", errResp.Code)
	}
}

func TestInvalidDewpointParam(t *testing.T) {
	h := New()
	req := httptest.NewRequest(http.MethodGet, "/api/juicecon?dewpoint=abc", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != "INVALID_PARAMS" {
		t.Errorf("expected error code INVALID_PARAMS, got %q", errResp.Code)
	}
}

func TestValidZIPWithMockClient(t *testing.T) {
	mock := &mockWeatherClient{
		observation: &weather.Observation{
			DewpointC:    21.1,
			DewpointF:    70.0,
			TemperatureF: 85.0,
			Timestamp:    time.Date(2026, 7, 15, 14, 0, 0, 0, time.UTC),
			Station:      "KJFK",
			City:         "New York",
			State:        "NY",
		},
	}
	h := NewWithClient(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/juicecon?zip=10001", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ActiveSystem != "juicecon" {
		t.Errorf("expected activeSystem juicecon, got %q", resp.ActiveSystem)
	}
	if resp.Level == nil || *resp.Level != 3 {
		t.Errorf("expected level 3 (JC3), got %v", resp.Level)
	}
	if resp.LevelDisplay != "JUICECON 3" {
		t.Errorf("expected levelDisplay JUICECON 3, got %q", resp.LevelDisplay)
	}
	if resp.Dewpoint != 70.0 {
		t.Errorf("expected dewpoint 70.0, got %f", resp.Dewpoint)
	}
	if resp.Temperature != 85.0 {
		t.Errorf("expected temperature 85.0, got %f", resp.Temperature)
	}
	if resp.Location.City != "New York" {
		t.Errorf("expected city New York, got %q", resp.Location.City)
	}
	if resp.Location.State != "NY" {
		t.Errorf("expected state NY, got %q", resp.Location.State)
	}
	if resp.Location.Station != "KJFK" {
		t.Errorf("expected station KJFK, got %q", resp.Location.Station)
	}
	if resp.Timestamp != "2026-07-15T14:00:00Z" {
		t.Errorf("expected timestamp 2026-07-15T14:00:00Z, got %q", resp.Timestamp)
	}
	if resp.AllClear {
		t.Error("expected allClear to be false")
	}
}
