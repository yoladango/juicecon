package weather

import "time"

// Observation represents weather observation data
type Observation struct {
	DewpointC    float64
	DewpointF    float64
	TemperatureF *float64
	Timestamp    time.Time
	Station      string
	City         string
	State        string

	// Condition is the human-readable weather description from NWS
	// (e.g. "Mostly Cloudy", "Light Rain", "Snow"). May be empty when
	// the station does not report a textDescription.
	Condition string

	// CloudCoverPct is the maximum cloud cover (0-100) reported by any
	// cloud layer in the observation. nil if not reported.
	CloudCoverPct *int

	// WindSpeedKmh is the reported sustained wind speed in km/h. nil if
	// not reported. Used by the client to drive precip/cloud drift speed.
	WindSpeedKmh *float64

	// IsDaytime is true when the observation timestamp falls between
	// roughly sunrise and sunset for the observation latitude. Computed
	// at the time of the request.
	IsDaytime bool
}

// NWSPointsResponse represents the NWS points API response
type NWSPointsResponse struct {
	Properties struct {
		RelativeLocation struct {
			Properties struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"properties"`
		} `json:"relativeLocation"`
		ObservationStations string `json:"observationStations"`
	} `json:"properties"`
}

// NWSStationsResponse represents the NWS stations list response
type NWSStationsResponse struct {
	Features []struct {
		Properties struct {
			StationIdentifier string `json:"stationIdentifier"`
		} `json:"properties"`
	} `json:"features"`
}

// NWSObservationResponse represents the NWS observation API response
type NWSObservationResponse struct {
	Properties struct {
		Temperature struct {
			Value    *float64 `json:"value"`
			UnitCode string   `json:"unitCode"`
		} `json:"temperature"`
		Dewpoint struct {
			Value    *float64 `json:"value"`
			UnitCode string   `json:"unitCode"`
		} `json:"dewpoint"`
		Timestamp       string `json:"timestamp"`
		TextDescription string `json:"textDescription"`
		WindSpeed       struct {
			Value    *float64 `json:"value"`
			UnitCode string   `json:"unitCode"`
		} `json:"windSpeed"`
		CloudLayers []struct {
			Amount string `json:"amount"`
		} `json:"cloudLayers"`
	} `json:"properties"`
}
