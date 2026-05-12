package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"juicecon-golang/internal/geo"
	"juicecon-golang/internal/index"
	"juicecon-golang/internal/weather"
)

var zipRegex = regexp.MustCompile(`^\d{5}$`)

// Response represents the unified API response
type Response struct {
	ActiveSystem  string   `json:"activeSystem"`
	SystemName    string   `json:"systemName"`
	Level         *int     `json:"level"`
	LevelDisplay  string   `json:"levelDisplay"`
	Dewpoint      float64  `json:"dewpoint"`
	Temperature   *float64 `json:"temperature"`
	Descriptor    string   `json:"descriptor"`
	Description   string   `json:"description"`
	Location      Location `json:"location"`
	Timestamp     string   `json:"timestamp"`
	AllClear      bool     `json:"allClear"`
	Condition     string   `json:"condition,omitempty"`
	CloudCoverPct *int     `json:"cloudCoverPct,omitempty"`
	WindSpeedKmh  *float64 `json:"windSpeedKmh,omitempty"`
	IsDaytime     bool     `json:"isDaytime"`
}

// Location represents location information
type Location struct {
	City    string `json:"city"`
	State   string `json:"state"`
	Station string `json:"station"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// WeatherClient defines the interface for fetching weather observations.
type WeatherClient interface {
	GetObservation(ctx context.Context, lat, lon float64) (*weather.Observation, error)
}

// Handler handles API requests
type Handler struct {
	weatherClient WeatherClient
}

// New creates a new API handler with a default weather client.
func New() *Handler {
	return &Handler{
		weatherClient: weather.NewClient(),
	}
}

// NewWithWeatherClient creates a handler with the given weather client,
// allowing the caller to share the client (and its cache) with other
// components such as the healthz endpoint.
func NewWithWeatherClient(client WeatherClient) *Handler {
	return &Handler{
		weatherClient: client,
	}
}

// NewWithClient creates a handler with a custom weather client (useful for testing).
func NewWithClient(client WeatherClient) *Handler {
	return &Handler{
		weatherClient: client,
	}
}

// ServeHTTP handles the /api/dewcon endpoint (also served at /api/juicecon for backward compat)
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	// Test mode: override dewpoint with ?dewpoint=XX to skip NWS API
	if dpStr := r.URL.Query().Get("dewpoint"); dpStr != "" {
		dp, err := strconv.ParseFloat(dpStr, 64)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid dewpoint value", "INVALID_PARAMS")
			return
		}
		h.writeTestResponse(w, dp, r.URL.Query())
		return
	}

	lat, lon, err := h.parseCoordinates(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error(), "INVALID_PARAMS")
		return
	}

	obs, err := h.weatherClient.GetObservation(r.Context(), lat, lon)
	if err != nil {
		slog.Error("weather API error",
			slog.Float64("lat", lat),
			slog.Float64("lon", lon),
			slog.String("error", err.Error()),
		)
		h.writeError(w, http.StatusBadGateway, "Unable to retrieve weather data. Please try again.", "WEATHER_API_ERROR")
		return
	}

	result := index.Evaluate(obs.DewpointF)

	resp := Response{
		ActiveSystem: string(result.ActiveSystem),
		SystemName:   result.SystemName,
		Level:        result.Level,
		LevelDisplay: result.LevelDisplay,
		Dewpoint:     obs.DewpointF,
		Temperature:  obs.TemperatureF,
		Descriptor:   result.Descriptor,
		Description:  result.Description,
		Location: Location{
			City:    obs.City,
			State:   obs.State,
			Station: obs.Station,
		},
		Timestamp:     obs.Timestamp.Format("2006-01-02T15:04:05Z"),
		AllClear:      result.AllClear,
		Condition:     obs.Condition,
		CloudCoverPct: obs.CloudCoverPct,
		WindSpeedKmh:  obs.WindSpeedKmh,
		IsDaytime:     obs.IsDaytime,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// writeTestResponse builds a response from a simulated dewpoint value.
// Test mode supports two optional query params for UI development:
//
//	?condition=...   override the textDescription (e.g. "Snow", "Rain")
//	?night=1         force IsDaytime to false
func (h *Handler) writeTestResponse(w http.ResponseWriter, dewpointF float64, q url.Values) {
	result := index.Evaluate(dewpointF)
	// Simulate a temperature roughly 10 degrees above the dewpoint
	tempF := math.Round((dewpointF+10)*10) / 10

	cond := q.Get("condition")
	day := true
	if v := q.Get("night"); v != "" && v != "0" {
		day = false
	}

	resp := Response{
		ActiveSystem: string(result.ActiveSystem),
		SystemName:   result.SystemName,
		Level:        result.Level,
		LevelDisplay: result.LevelDisplay,
		Dewpoint:     math.Round(dewpointF*10) / 10,
		Temperature:  &tempF,
		Descriptor:   result.Descriptor,
		Description:  result.Description,
		Location: Location{
			City:    "Test Mode",
			State:   "SIM",
			Station: "KTEST",
		},
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		AllClear:  result.AllClear,
		Condition: cond,
		IsDaytime: day,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) parseCoordinates(r *http.Request) (float64, float64, error) {
	query := r.URL.Query()

	// Check for ZIP code first
	if zip := query.Get("zip"); zip != "" {
		if !zipRegex.MatchString(zip) {
			return 0, 0, &paramError{"Invalid ZIP code: must be exactly 5 digits"}
		}
		coords, err := geo.LookupZIP(zip)
		if err != nil {
			return 0, 0, &paramError{"ZIP code not found"}
		}
		return coords.Lat, coords.Lon, nil
	}

	// Otherwise, require lat/lon
	latStr := query.Get("lat")
	lonStr := query.Get("lon")

	if latStr == "" || lonStr == "" {
		return 0, 0, &paramError{"Must provide either lat/lon or zip parameter"}
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, &paramError{"Invalid latitude"}
	}
	if lat < -90 || lat > 90 {
		return 0, 0, &paramError{fmt.Sprintf("Latitude must be between -90 and 90, got %.4f", lat)}
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, &paramError{"Invalid longitude"}
	}
	if lon < -180 || lon > 180 {
		return 0, 0, &paramError{fmt.Sprintf("Longitude must be between -180 and 180, got %.4f", lon)}
	}

	return lat, lon, nil
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message, code string) {
	h.writeJSON(w, status, ErrorResponse{Error: message, Code: code})
}

type paramError struct {
	message string
}

func (e *paramError) Error() string {
	return e.message
}
