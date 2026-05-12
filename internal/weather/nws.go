package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
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
	// Fall back to "now" when NWS omits the timestamp, so day/night detection
	// still has something reasonable to work with.
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	// Wind speed: NWS reports in km/h ("wmoUnit:km_h-1") or sometimes m/s.
	var windKmh *float64
	if obs.Properties.WindSpeed.Value != nil {
		v := *obs.Properties.WindSpeed.Value
		if strings.Contains(obs.Properties.WindSpeed.UnitCode, "m_s") {
			v = v * 3.6
		}
		windKmh = &v
	}

	// Cloud cover: take the densest reported layer. NWS uses METAR codes.
	cloudCodes := make([]string, 0, len(obs.Properties.CloudLayers))
	for _, l := range obs.Properties.CloudLayers {
		cloudCodes = append(cloudCodes, l.Amount)
	}
	cloudPct := cloudLayersToPct(cloudCodes)

	result := &Observation{
		DewpointC:     dewpointC,
		DewpointF:     dewpointF,
		TemperatureF:  temperatureF,
		Timestamp:     timestamp,
		Station:       stationID,
		City:          points.Properties.RelativeLocation.Properties.City,
		State:         points.Properties.RelativeLocation.Properties.State,
		Condition:     strings.TrimSpace(obs.Properties.TextDescription),
		CloudCoverPct: cloudPct,
		WindSpeedKmh:  windKmh,
		IsDaytime:     isDaytime(lat, lon, timestamp),
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

// cloudLayersToPct returns the densest cloud cover percentage represented
// by the slice of METAR cloud-amount codes ("SKC", "CLR", "FEW", "SCT",
// "BKN", "OVC", "VV"). Unknown codes are ignored. nil is returned when
// no useful information is present.
func cloudLayersToPct(codes []string) *int {
	if len(codes) == 0 {
		return nil
	}
	best := -1
	for _, raw := range codes {
		c := strings.ToUpper(strings.TrimSpace(raw))
		var pct int
		switch c {
		case "SKC", "CLR", "NCD", "NSC":
			pct = 0
		case "FEW":
			pct = 20
		case "SCT":
			pct = 45
		case "BKN":
			pct = 75
		case "OVC", "VV":
			pct = 100
		default:
			continue
		}
		if pct > best {
			best = pct
		}
	}
	if best < 0 {
		return nil
	}
	return &best
}

// isDaytime returns true when the sun is above the horizon at the given
// latitude/longitude and instant (UTC). Uses a NOAA-derived solar
// position approximation accurate to within a couple of minutes — more
// than enough to switch a day/night background.
func isDaytime(lat, lon float64, t time.Time) bool {
	return solarElevationDeg(lat, lon, t) > 0
}

func solarElevationDeg(lat, lon float64, t time.Time) float64 {
	tUTC := t.UTC()
	doy := float64(tUTC.YearDay())
	utcHours := float64(tUTC.Hour()) + float64(tUTC.Minute())/60 + float64(tUTC.Second())/3600

	// Fractional year (radians)
	y := 2 * math.Pi / 365 * (doy - 1 + (utcHours-12)/24)

	// Equation of time (minutes)
	eot := 229.18 * (0.000075 +
		0.001868*math.Cos(y) -
		0.032077*math.Sin(y) -
		0.014615*math.Cos(2*y) -
		0.040849*math.Sin(2*y))

	// Solar declination (radians)
	decl := 0.006918 -
		0.399912*math.Cos(y) +
		0.070257*math.Sin(y) -
		0.006758*math.Cos(2*y) +
		0.000907*math.Sin(2*y) -
		0.002697*math.Cos(3*y) +
		0.00148*math.Sin(3*y)

	// True solar time (minutes)
	tst := utcHours*60 + eot + 4*lon

	// Hour angle (radians); 720 min = solar noon
	ha := (tst/4 - 180) * math.Pi / 180

	latRad := lat * math.Pi / 180

	sinElev := math.Sin(latRad)*math.Sin(decl) +
		math.Cos(latRad)*math.Cos(decl)*math.Cos(ha)
	// Clamp for safety
	if sinElev > 1 {
		sinElev = 1
	} else if sinElev < -1 {
		sinElev = -1
	}
	return math.Asin(sinElev) * 180 / math.Pi
}
