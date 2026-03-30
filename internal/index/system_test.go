package index

import "testing"

func intPtr(i int) *int {
	return &i
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name             string
		dewpointF        float64
		wantActiveSystem SystemID
		wantSystemName   string
		wantLevel        *int
		wantLevelDisplay string
		wantDescriptor   string
		wantDescription  string
		wantAllClear     bool
	}{
		// --- JuiceCon activation (dewpoint >= 60) ---

		// JC5 at exact boundary 60.0
		{
			name:             "JuiceCon activates at 60.0 (JC5 boundary)",
			dewpointF:        60.0,
			wantActiveSystem: SystemJuiceCon,
			wantSystemName:   "JUICECON",
			wantLevel:        intPtr(5),
			wantLevelDisplay: "JUICECON 5",
			wantDescriptor:   "Noticeable",
			wantDescription:  "A/C at night is now justified.",
			wantAllClear:     false,
		},
		// JC1 at 75.0
		{
			name:             "JuiceCon JC1 at 75.0",
			dewpointF:        75.0,
			wantActiveSystem: SystemJuiceCon,
			wantSystemName:   "JUICECON",
			wantLevel:        intPtr(1),
			wantLevelDisplay: "JUICECON 1",
			wantDescriptor:   "The Ultimate",
			wantDescription:  "A very rare event. This is not a drill.",
			wantAllClear:     false,
		},

		// --- CCF activation (dewpoint <= 25) ---

		// CCF5 at exact boundary 25.0
		{
			name:             "CCF activates at 25.0 (CCF5 boundary)",
			dewpointF:        25.0,
			wantActiveSystem: SystemCCF,
			wantSystemName:   "CCF",
			wantLevel:        intPtr(5),
			wantLevelDisplay: "CCF 5",
			wantDescriptor:   "Cotton Mouth",
			wantDescription:  "Cotton mouth but no alcohol? Hmmmm. Saline nasal rinse under consideration.",
			wantAllClear:     false,
		},
		// CCF1 at 2.0
		{
			name:             "CCF1 at 2.0",
			dewpointF:        2.0,
			wantActiveSystem: SystemCCF,
			wantSystemName:   "CCF",
			wantLevel:        intPtr(1),
			wantLevelDisplay: "CCF 1",
			wantDescriptor:   "Walking EMP",
			wantDescription:  "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!",
			wantAllClear:     false,
		},
		// CCF1 at negative dewpoint
		{
			name:             "CCF1 at -10.0",
			dewpointF:        -10.0,
			wantActiveSystem: SystemCCF,
			wantSystemName:   "CCF",
			wantLevel:        intPtr(1),
			wantLevelDisplay: "CCF 1",
			wantDescriptor:   "Walking EMP",
			wantDescription:  "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!",
			wantAllClear:     false,
		},

		// --- Comfort zone (26-59°F, neither system active) ---

		{
			name:             "Comfort zone at 26.0",
			dewpointF:        26.0,
			wantActiveSystem: SystemNone,
			wantSystemName:   "",
			wantLevel:        nil,
			wantLevelDisplay: "ALL CLEAR",
			wantDescriptor:   "Comfortable",
			wantDescription:  "All atmospheric moisture protocols inactive. Conditions nominal.",
			wantAllClear:     true,
		},
		{
			name:             "Comfort zone at 40.0",
			dewpointF:        40.0,
			wantActiveSystem: SystemNone,
			wantSystemName:   "",
			wantLevel:        nil,
			wantLevelDisplay: "ALL CLEAR",
			wantDescriptor:   "Comfortable",
			wantDescription:  "All atmospheric moisture protocols inactive. Conditions nominal.",
			wantAllClear:     true,
		},
		{
			name:             "Comfort zone at 59.0",
			dewpointF:        59.0,
			wantActiveSystem: SystemNone,
			wantSystemName:   "",
			wantLevel:        nil,
			wantLevelDisplay: "ALL CLEAR",
			wantDescriptor:   "Comfortable",
			wantDescription:  "All atmospheric moisture protocols inactive. Conditions nominal.",
			wantAllClear:     true,
		},
		{
			name:             "Comfort zone at 59.9",
			dewpointF:        59.9,
			wantActiveSystem: SystemNone,
			wantSystemName:   "",
			wantLevel:        nil,
			wantLevelDisplay: "ALL CLEAR",
			wantDescriptor:   "Comfortable",
			wantDescription:  "All atmospheric moisture protocols inactive. Conditions nominal.",
			wantAllClear:     true,
		},

		// --- Boundary transitions ---

		// CCF/Comfort boundary: 25.0 -> CCF, 25.1 -> Comfort
		{
			name:             "Boundary: 25.0 routes to CCF",
			dewpointF:        25.0,
			wantActiveSystem: SystemCCF,
			wantSystemName:   "CCF",
			wantLevel:        intPtr(5),
			wantLevelDisplay: "CCF 5",
			wantDescriptor:   "Cotton Mouth",
			wantDescription:  "Cotton mouth but no alcohol? Hmmmm. Saline nasal rinse under consideration.",
			wantAllClear:     false,
		},
		{
			name:             "Boundary: 25.1 routes to Comfort zone",
			dewpointF:        25.1,
			wantActiveSystem: SystemNone,
			wantSystemName:   "",
			wantLevel:        nil,
			wantLevelDisplay: "ALL CLEAR",
			wantDescriptor:   "Comfortable",
			wantDescription:  "All atmospheric moisture protocols inactive. Conditions nominal.",
			wantAllClear:     true,
		},
		// Comfort/JuiceCon boundary: 59.9 -> Comfort, 60.0 -> JuiceCon
		{
			name:             "Boundary: 59.9 routes to Comfort zone",
			dewpointF:        59.9,
			wantActiveSystem: SystemNone,
			wantSystemName:   "",
			wantLevel:        nil,
			wantLevelDisplay: "ALL CLEAR",
			wantDescriptor:   "Comfortable",
			wantDescription:  "All atmospheric moisture protocols inactive. Conditions nominal.",
			wantAllClear:     true,
		},
		{
			name:             "Boundary: 60.0 routes to JuiceCon",
			dewpointF:        60.0,
			wantActiveSystem: SystemJuiceCon,
			wantSystemName:   "JUICECON",
			wantLevel:        intPtr(5),
			wantLevelDisplay: "JUICECON 5",
			wantDescriptor:   "Noticeable",
			wantDescription:  "A/C at night is now justified.",
			wantAllClear:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.dewpointF)

			if got.ActiveSystem != tt.wantActiveSystem {
				t.Errorf("ActiveSystem = %q, want %q", got.ActiveSystem, tt.wantActiveSystem)
			}

			if got.SystemName != tt.wantSystemName {
				t.Errorf("SystemName = %q, want %q", got.SystemName, tt.wantSystemName)
			}

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

			if got.LevelDisplay != tt.wantLevelDisplay {
				t.Errorf("LevelDisplay = %q, want %q", got.LevelDisplay, tt.wantLevelDisplay)
			}

			if got.Descriptor != tt.wantDescriptor {
				t.Errorf("Descriptor = %q, want %q", got.Descriptor, tt.wantDescriptor)
			}

			if got.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDescription)
			}

			if got.AllClear != tt.wantAllClear {
				t.Errorf("AllClear = %v, want %v", got.AllClear, tt.wantAllClear)
			}
		})
	}
}
