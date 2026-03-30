package ccf

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
		wantDescrip string
		wantClear   bool
	}{
		// CCF1: <= 2°F
		{"CCF1 at boundary 2.0", 2.0, ptr.Int(1), "Walking EMP", "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!", false},
		{"CCF1 at zero", 0.0, ptr.Int(1), "Walking EMP", "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!", false},
		{"CCF1 at -10", -10.0, ptr.Int(1), "Walking EMP", "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!", false},
		{"CCF1 at extreme -40", -40.0, ptr.Int(1), "Walking EMP", "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!", false},

		// CCF2: 3-8°F
		{"CCF2 at lower boundary 3.0", 3.0, ptr.Int(2), "Husk Hands", "Hands are husks of their former selves.", false},
		{"CCF2 at upper boundary 8.0", 8.0, ptr.Int(2), "Husk Hands", "Hands are husks of their former selves.", false},
		{"CCF2 at midpoint 5.0", 5.0, ptr.Int(2), "Husk Hands", "Hands are husks of their former selves.", false},

		// CCF3: 9-14°F
		{"CCF3 at lower boundary 9.0", 9.0, ptr.Int(3), "Lotion Failure", "This lotion is NOT cutting it.", false},
		{"CCF3 at upper boundary 14.0", 14.0, ptr.Int(3), "Lotion Failure", "This lotion is NOT cutting it.", false},
		{"CCF3 at midpoint 11.0", 11.0, ptr.Int(3), "Lotion Failure", "This lotion is NOT cutting it.", false},

		// CCF4: 15-19°F
		{"CCF4 at lower boundary 15.0", 15.0, ptr.Int(4), "Humidifier Check", "Do we even have a working humidifier?", false},
		{"CCF4 at upper boundary 19.0", 19.0, ptr.Int(4), "Humidifier Check", "Do we even have a working humidifier?", false},
		{"CCF4 at midpoint 17.0", 17.0, ptr.Int(4), "Humidifier Check", "Do we even have a working humidifier?", false},

		// CCF5: 20-25°F
		{"CCF5 at lower boundary 20.0", 20.0, ptr.Int(5), "Cotton Mouth", "Cotton mouth but no alcohol? Hmmmm. Saline nasal rinse under consideration.", false},
		{"CCF5 at upper boundary 25.0", 25.0, ptr.Int(5), "Cotton Mouth", "Cotton mouth but no alcohol? Hmmmm. Saline nasal rinse under consideration.", false},
		{"CCF5 at midpoint 22.0", 22.0, ptr.Int(5), "Cotton Mouth", "Cotton mouth but no alcohol? Hmmmm. Saline nasal rinse under consideration.", false},

		// ALL CLEAR: > 25°F
		{"ALL CLEAR just above boundary 25.1", 25.1, nil, "Comfortable", "CCF protocols not currently active.", true},
		{"ALL CLEAR at 26.0", 26.0, nil, "Comfortable", "CCF protocols not currently active.", true},
		{"ALL CLEAR at 50.0", 50.0, nil, "Comfortable", "CCF protocols not currently active.", true},
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

			if got.Description != tt.wantDescrip {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDescrip)
			}

			if got.AllClear != tt.wantClear {
				t.Errorf("AllClear = %v, want %v", got.AllClear, tt.wantClear)
			}
		})
	}
}

func TestLevelDisplay(t *testing.T) {
	tests := []struct {
		name string
		level Level
		want  string
	}{
		{"CCF1 display", Level{Level: ptr.Int(1), AllClear: false}, "CCF 1"},
		{"CCF2 display", Level{Level: ptr.Int(2), AllClear: false}, "CCF 2"},
		{"CCF3 display", Level{Level: ptr.Int(3), AllClear: false}, "CCF 3"},
		{"CCF4 display", Level{Level: ptr.Int(4), AllClear: false}, "CCF 4"},
		{"CCF5 display", Level{Level: ptr.Int(5), AllClear: false}, "CCF 5"},
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
