package handler

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"juicecon-golang/internal/geo"
	"juicecon-golang/internal/index"
	"juicecon-golang/internal/weather"
)

// Response represents the unified API response
type Response struct {
	ActiveSystem string   `json:"activeSystem"`
	SystemName   string   `json:"systemName"`
	Level        *int     `json:"level"`
	LevelDisplay string   `json:"levelDisplay"`
	Dewpoint     float64  `json:"dewpoint"`
	Temperature  *float64 `json:"temperature"`
	Descriptor   string   `json:"descriptor"`
	Description  string   `json:"description"`
	Location     Location `json:"location"`
	Timestamp    string   `json:"timestamp"`
	AllClear     bool     `json:"allClear"`
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

// New creates a new API handler
func New() *Handler {
	return &Handler{
		weatherClient: weather.NewClient(),
	}
}

// NewWithClient creates a handler with a custom weather client (useful for testing).
func NewWithClient(client WeatherClient) *Handler {
	return &Handler{
		weatherClient: client,
	}
}

// ServeHTTP handles the /api/juicecon endpoint
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
		h.writeTestResponse(w, dp)
		return
	}

	lat, lon, err := h.parseCoordinates(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error(), "INVALID_PARAMS")
		return
	}

	obs, err := h.weatherClient.GetObservation(r.Context(), lat, lon)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, "Unable to fetch weather data: "+err.Error(), "WEATHER_API_ERROR")
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
		Timestamp: obs.Timestamp.Format("2006-01-02T15:04:05Z"),
		AllClear:  result.AllClear,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// writeTestResponse builds a response from a simulated dewpoint value
func (h *Handler) writeTestResponse(w http.ResponseWriter, dewpointF float64) {
	result := index.Evaluate(dewpointF)
	// Simulate a temperature roughly 10 degrees above the dewpoint
	tempF := math.Round((dewpointF+10)*10) / 10

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
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) parseCoordinates(r *http.Request) (float64, float64, error) {
	query := r.URL.Query()

	// Check for ZIP code first
	if zip := query.Get("zip"); zip != "" {
		coords, err := geo.LookupZIP(zip)
		if err != nil {
			return 0, 0, err
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

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, &paramError{"Invalid longitude"}
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
