package geo

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed zips.json
var zipsData []byte

// Coordinates represents a lat/lon pair
type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

var zipLookup map[string]Coordinates

func init() {
	zipLookup = make(map[string]Coordinates)
	if err := json.Unmarshal(zipsData, &zipLookup); err != nil {
		panic(fmt.Sprintf("failed to load ZIP data: %v", err))
	}
}

// LookupZIP returns coordinates for a US ZIP code
func LookupZIP(zip string) (Coordinates, error) {
	coords, ok := zipLookup[zip]
	if !ok {
		return Coordinates{}, fmt.Errorf("ZIP code not found: %s", zip)
	}
	return coords, nil
}
