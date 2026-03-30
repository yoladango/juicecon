package weather

import (
	"math"
	"testing"
)

func TestCelsiusToFahrenheit(t *testing.T) {
	tests := []struct {
		name    string
		celsius float64
		wantF   float64
	}{
		{
			name:    "freezing point",
			celsius: 0,
			wantF:   32,
		},
		{
			name:    "boiling point",
			celsius: 100,
			wantF:   212,
		},
		{
			name:    "negative 40 crossover",
			celsius: -40,
			wantF:   -40,
		},
		{
			name:    "body temperature",
			celsius: 37,
			wantF:   98.6,
		},
		{
			name:    "sub-zero celsius",
			celsius: -20,
			wantF:   -4,
		},
		{
			name:    "mild temperature",
			celsius: 20,
			wantF:   68,
		},
	}

	const tolerance = 0.01

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := celsiusToFahrenheit(tt.celsius)
			if math.Abs(got-tt.wantF) > tolerance {
				t.Errorf("celsiusToFahrenheit(%g) = %g, want %g", tt.celsius, got, tt.wantF)
			}
		})
	}
}
