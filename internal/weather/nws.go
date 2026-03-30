package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	nwsBaseURL  = "https://api.weather.gov"
	userAgent   = "(juicecon.app, contact@juicecon.app)"
	httpTimeout = 10 * time.Second
)

// HTTPDoer abstracts HTTP request execution for testability.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client handles NWS API requests
type Client struct {
	httpClient HTTPDoer
	cache      *observationCache
}

// NewClient creates a new NWS API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		cache: newObservationCache(defaultCacheTTL),
	}
}

// CacheStats returns metrics about the observation cache.
func (c *Client) CacheStats() CacheStats {
	return c.cache.stats()
}

// GetObservation fetches the current weather observation for a location.
// Results are cached in memory; repeated calls for the same (rounded) lat/lon
// within the TTL window are served from cache without hitting the NWS API.
func (c *Client) GetObservation(ctx context.Context, lat, lon float64) (*Observation, error) {
	key := cacheKey(lat, lon)
	if cached, ok := c.cache.get(key); ok {
		return cached, nil
	}

	// Step 1: Get the points data to find station and location info
	pointsURL := fmt.Sprintf("%s/points/%.4f,%.4f", nwsBaseURL, lat, lon)
	pointsResp, err := c.makeRequest(ctx, pointsURL)
	if err != nil {
		return nil, fmt.Errorf("points lookup failed: %w", err)
	}
	defer pointsResp.Body.Close()

	var points NWSPointsResponse
	if err := json.NewDecoder(pointsResp.Body).Decode(&points); err != nil {
		return nil, fmt.Errorf("failed to decode points response: %w", err)
	}

	// Step 2: Get the stations list to find the nearest station
	stationsResp, err := c.makeRequest(ctx, points.Properties.ObservationStations)
	if err != nil {
		return nil, fmt.Errorf("stations lookup failed: %w", err)
	}
	defer stationsResp.Body.Close()

	var stations NWSStationsResponse
	if err := json.NewDecoder(stationsResp.Body).Decode(&stations); err != nil {
		return nil, fmt.Errorf("failed to decode stations response: %w", err)
	}

	if len(stations.Features) == 0 {
		return nil, fmt.Errorf("no observation stations found")
	}

	// Step 3: Try stations in order until one returns a valid observation
	var obs NWSObservationResponse
	var stationID string
	maxStations := 5
	if len(stations.Features) < maxStations {
		maxStations = len(stations.Features)
	}

	var lastErr error
	for i := 0; i < maxStations; i++ {
		stationID = stations.Features[i].Properties.StationIdentifier
		obsURL := fmt.Sprintf("%s/stations/%s/observations/latest", nwsBaseURL, stationID)
		obsResp, err := c.makeRequest(ctx, obsURL)
		if err != nil {
			lastErr = err
			continue
		}

		if err := json.NewDecoder(obsResp.Body).Decode(&obs); err != nil {
			obsResp.Body.Close()
			lastErr = err
			continue
		}
		obsResp.Body.Close()

		if obs.Properties.Dewpoint.Value != nil {
			lastErr = nil
			break
		}
		lastErr = fmt.Errorf("dewpoint data not available from station %s", stationID)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("observation lookup failed: %w", lastErr)
	}

	dewpointC := *obs.Properties.Dewpoint.Value
	dewpointF := celsiusToFahrenheit(dewpointC)

	var temperatureF *float64
	if obs.Properties.Temperature.Value != nil {
		f := celsiusToFahrenheit(*obs.Properties.Temperature.Value)
		temperatureF = &f
	}

	timestamp, _ := time.Parse(time.RFC3339, obs.Properties.Timestamp)

	result := &Observation{
		DewpointC:    dewpointC,
		DewpointF:    dewpointF,
		TemperatureF: temperatureF,
		Timestamp:    timestamp,
		Station:      stationID,
		City:         points.Properties.RelativeLocation.Properties.City,
		State:        points.Properties.RelativeLocation.Properties.State,
	}

	c.cache.set(key, result)

	return result, nil
}

func (c *Client) makeRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return resp, nil
}

func celsiusToFahrenheit(c float64) float64 {
	return (c * 9 / 5) + 32
}
