package weather

import (
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

// Client handles NWS API requests
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new NWS API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// GetObservation fetches the current weather observation for a location
func (c *Client) GetObservation(lat, lon float64) (*Observation, error) {
	// Step 1: Get the points data to find station and location info
	pointsURL := fmt.Sprintf("%s/points/%.4f,%.4f", nwsBaseURL, lat, lon)
	pointsResp, err := c.makeRequest(pointsURL)
	if err != nil {
		return nil, fmt.Errorf("points lookup failed: %w", err)
	}
	defer pointsResp.Body.Close()

	var points NWSPointsResponse
	if err := json.NewDecoder(pointsResp.Body).Decode(&points); err != nil {
		return nil, fmt.Errorf("failed to decode points response: %w", err)
	}

	// Step 2: Get the stations list to find the nearest station
	stationsResp, err := c.makeRequest(points.Properties.ObservationStations)
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

	stationID := stations.Features[0].Properties.StationIdentifier

	// Step 3: Get the latest observation from the nearest station
	obsURL := fmt.Sprintf("%s/stations/%s/observations/latest", nwsBaseURL, stationID)
	obsResp, err := c.makeRequest(obsURL)
	if err != nil {
		return nil, fmt.Errorf("observation lookup failed: %w", err)
	}
	defer obsResp.Body.Close()

	var obs NWSObservationResponse
	if err := json.NewDecoder(obsResp.Body).Decode(&obs); err != nil {
		return nil, fmt.Errorf("failed to decode observation response: %w", err)
	}

	if obs.Properties.Dewpoint.Value == nil {
		return nil, fmt.Errorf("dewpoint data not available")
	}

	dewpointC := *obs.Properties.Dewpoint.Value
	dewpointF := celsiusToFahrenheit(dewpointC)

	timestamp, _ := time.Parse(time.RFC3339, obs.Properties.Timestamp)

	return &Observation{
		DewpointC: dewpointC,
		DewpointF: dewpointF,
		Timestamp: timestamp,
		Station:   stationID,
		City:      points.Properties.RelativeLocation.Properties.City,
		State:     points.Properties.RelativeLocation.Properties.State,
	}, nil
}

func (c *Client) makeRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
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
