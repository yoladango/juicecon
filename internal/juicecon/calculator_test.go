package juicecon

import (
	"testing"

	"juicecon-golang/internal/ptr"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name        string
		dewpointF   float64
		wantLevel   *int
		wantDesc    string
		wantDescr   string
		wantAllClr  bool
	}{
		// JC1: >= 75
		{"JC1 at boundary 75.0", 75.0, ptr.Int(1), "The Ultimate", "A very rare event. This is not a drill.", false},
		{"JC1 extreme high 100.0", 100.0, ptr.Int(1), "The Ultimate", "A very rare event. This is not a drill.", false},
		{"JC1 extreme high 150.0", 150.0, ptr.Int(1), "The Ultimate", "A very rare event. This is not a drill.", false},

		// JC2: 73-74.9
		{"JC2 at boundary 73.0", 73.0, ptr.Int(2), "Come The Fuck On", "Unacceptable. File complaints with the atmosphere.", false},
		{"JC2 at 74.9", 74.9, ptr.Int(2), "Come The Fuck On", "Unacceptable. File complaints with the atmosphere.", false},
		{"JC2 at 74.99", 74.99, ptr.Int(2), "Come The Fuck On", "Unacceptable. File complaints with the atmosphere.", false},

		// JC3: 70-72.9
		{"JC3 at boundary 70.0", 70.0, ptr.Int(3), "Unbearable", "The air has weight. You are breathing soup.", false},
		{"JC3 at 72.9", 72.9, ptr.Int(3), "Unbearable", "The air has weight. You are breathing soup.", false},

		// JC4: 65-69.9
		{"JC4 at boundary 65.0", 65.0, ptr.Int(4), "Miserable", "Existence is damp. Consider relocation.", false},
		{"JC4 at 69.9", 69.9, ptr.Int(4), "Miserable", "Existence is damp. Consider relocation.", false},

		// JC5: 60-64.9
		{"JC5 at boundary 60.0", 60.0, ptr.Int(5), "Noticeable", "A/C at night is now justified.", false},
		{"JC5 at 64.9", 64.9, ptr.Int(5), "Noticeable", "A/C at night is now justified.", false},

		// ALL CLEAR: < 60
		{"ALL CLEAR just below 59.9", 59.9, nil, "Comfortable", "JUICECON protocols not currently active.", true},
		{"ALL CLEAR at 59.0", 59.0, nil, "Comfortable", "JUICECON protocols not currently active.", true},
		{"ALL CLEAR at 0.0", 0.0, nil, "Comfortable", "JUICECON protocols not currently active.", true},

		// Negative dewpoints
		{"ALL CLEAR negative -10.0", -10.0, nil, "Comfortable", "JUICECON protocols not currently active.", true},
		{"ALL CLEAR negative -40.0", -40.0, nil, "Comfortable", "JUICECON protocols not currently active.", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Calculate(tt.dewpointF)

			// Check Level pointer
			if tt.wantLevel == nil {
				if got.Level != nil {
					t.Errorf("Level = %d, want nil", *got.Level)
				}
			} else {
				if got.Level == nil {
					t.Fatalf("Level = nil, want %d", *tt.wantLevel)
				}
				if *got.Level != *tt.wantLevel {
					t.Errorf("Level = %d, want %d", *got.Level, *tt.wantLevel)
				}
			}

			if got.Descriptor != tt.wantDesc {
				t.Errorf("Descriptor = %q, want %q", got.Descriptor, tt.wantDesc)
			}

			if got.Description != tt.wantDescr {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDescr)
			}

			if got.AllClear != tt.wantAllClr {
				t.Errorf("AllClear = %v, want %v", got.AllClear, tt.wantAllClr)
			}
		})
	}
}

func TestLevelDisplay(t *testing.T) {
	tests := []struct {
		name    string
		level   Level
		want    string
	}{
		{"JC1 display", Level{Level: ptr.Int(1), AllClear: false}, "JUICECON 1"},
		{"JC2 display", Level{Level: ptr.Int(2), AllClear: false}, "JUICECON 2"},
		{"JC3 display", Level{Level: ptr.Int(3), AllClear: false}, "JUICECON 3"},
		{"JC4 display", Level{Level: ptr.Int(4), AllClear: false}, "JUICECON 4"},
		{"JC5 display", Level{Level: ptr.Int(5), AllClear: false}, "JUICECON 5"},
		{"ALL CLEAR display", Level{Level: nil, AllClear: true}, "ALL CLEAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.LevelDisplay()
			if got != tt.want {
				t.Errorf("LevelDisplay() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCalculateLevelDisplay(t *testing.T) {
	tests := []struct {
		name      string
		dewpointF float64
		want      string
	}{
		{"JC1 via Calculate", 75.0, "JUICECON 1"},
		{"JC2 via Calculate", 73.0, "JUICECON 2"},
		{"JC3 via Calculate", 70.0, "JUICECON 3"},
		{"JC4 via Calculate", 65.0, "JUICECON 4"},
		{"JC5 via Calculate", 60.0, "JUICECON 5"},
		{"ALL CLEAR via Calculate", 59.0, "ALL CLEAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := Calculate(tt.dewpointF)
			got := level.LevelDisplay()
			if got != tt.want {
				t.Errorf("Calculate(%v).LevelDisplay() = %q, want %q", tt.dewpointF, got, tt.want)
			}
		})
	}
}
