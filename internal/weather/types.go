package weather

import "time"

// Observation represents weather observation data
type Observation struct {
	DewpointC float64
	DewpointF float64
	Timestamp time.Time
	Station   string
	City      string
	State     string
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
		Dewpoint struct {
			Value    *float64 `json:"value"`
			UnitCode string   `json:"unitCode"`
		} `json:"dewpoint"`
		Timestamp string `json:"timestamp"`
	} `json:"properties"`
}
