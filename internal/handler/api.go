package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"juicecon-golang/internal/geo"
	"juicecon-golang/internal/juicecon"
	"juicecon-golang/internal/weather"
)

// Response represents the API response
type Response struct {
	Level        *int     `json:"level"`
	LevelDisplay string   `json:"levelDisplay"`
	Dewpoint     float64  `json:"dewpoint"`
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

// Handler handles API requests
type Handler struct {
	weatherClient *weather.Client
}

// New creates a new API handler
func New() *Handler {
	return &Handler{
		weatherClient: weather.NewClient(),
	}
}

// ServeHTTP handles the /api/juicecon endpoint
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	lat, lon, err := h.parseCoordinates(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error(), "INVALID_PARAMS")
		return
	}

	obs, err := h.weatherClient.GetObservation(lat, lon)
	if err != nil {
		h.writeError(w, http.StatusBadGateway, "Unable to fetch weather data: "+err.Error(), "WEATHER_API_ERROR")
		return
	}

	level := juicecon.Calculate(obs.DewpointF)

	resp := Response{
		Level:        level.Level,
		LevelDisplay: level.LevelDisplay(),
		Dewpoint:     obs.DewpointF,
		Descriptor:   level.Descriptor,
		Description:  level.Description,
		Location: Location{
			City:    obs.City,
			State:   obs.State,
			Station: obs.Station,
		},
		Timestamp: obs.Timestamp.Format("2006-01-02T15:04:05Z"),
		AllClear:  level.AllClear,
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
