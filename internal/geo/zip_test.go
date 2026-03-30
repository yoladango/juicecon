package geo

import (
	"math"
	"testing"
)

func TestLookupZIP(t *testing.T) {
	tests := []struct {
		name      string
		zip       string
		wantLat   float64
		wantLon   float64
		wantError bool
	}{
		{
			name:      "valid NYC ZIP 10001",
			zip:       "10001",
			wantLat:   40.7484,
			wantLon:   -73.9967,
			wantError: false,
		},
		{
			name:      "nonexistent ZIP 00000",
			zip:       "00000",
			wantError: true,
		},
		{
			name:      "empty string",
			zip:       "",
			wantError: true,
		},
		{
			name:      "non-numeric input",
			zip:       "abcde",
			wantError: true,
		},
		{
			name:      "four-digit string",
			zip:       "1000",
			wantError: true,
		},
		{
			name:      "six-digit string",
			zip:       "100011",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coords, err := LookupZIP(tt.zip)

			if tt.wantError {
				if err == nil {
					t.Fatalf("LookupZIP(%q) expected error, got nil", tt.zip)
				}
				return
			}

			if err != nil {
				t.Fatalf("LookupZIP(%q) unexpected error: %v", tt.zip, err)
			}

			const tolerance = 0.001
			if math.Abs(coords.Lat-tt.wantLat) > tolerance {
				t.Errorf("LookupZIP(%q) lat = %f, want %f", tt.zip, coords.Lat, tt.wantLat)
			}
			if math.Abs(coords.Lon-tt.wantLon) > tolerance {
				t.Errorf("LookupZIP(%q) lon = %f, want %f", tt.zip, coords.Lon, tt.wantLon)
			}

			// Sanity check: latitude and longitude within US bounds
			if coords.Lat < 24 || coords.Lat > 72 {
				t.Errorf("LookupZIP(%q) lat %f outside reasonable US range [24, 72]", tt.zip, coords.Lat)
			}
			if coords.Lon < -180 || coords.Lon > -65 {
				t.Errorf("LookupZIP(%q) lon %f outside reasonable US range [-180, -65]", tt.zip, coords.Lon)
			}
		})
	}
}
