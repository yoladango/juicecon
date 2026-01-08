package juicecon

import "strconv"

// Level represents a JUICECON severity level
type Level struct {
	Level       *int   `json:"level"`
	Descriptor  string `json:"descriptor"`
	Description string `json:"description"`
	AllClear    bool   `json:"allClear"`
}

// Calculate determines the JUICECON level from a dewpoint in Fahrenheit
func Calculate(dewpointF float64) Level {
	switch {
	case dewpointF >= 75:
		return Level{
			Level:       intPtr(1),
			Descriptor:  "The Ultimate",
			Description: "A very rare event. This is not a drill.",
			AllClear:    false,
		}
	case dewpointF >= 73:
		return Level{
			Level:       intPtr(2),
			Descriptor:  "Come The Fuck On",
			Description: "Unacceptable. File complaints with the atmosphere.",
			AllClear:    false,
		}
	case dewpointF >= 70:
		return Level{
			Level:       intPtr(3),
			Descriptor:  "Unbearable",
			Description: "The air has weight. You are breathing soup.",
			AllClear:    false,
		}
	case dewpointF >= 65:
		return Level{
			Level:       intPtr(4),
			Descriptor:  "Miserable",
			Description: "Existence is damp. Consider relocation.",
			AllClear:    false,
		}
	case dewpointF >= 60:
		return Level{
			Level:       intPtr(5),
			Descriptor:  "Noticeable",
			Description: "A/C at night is now justified.",
			AllClear:    false,
		}
	default:
		return Level{
			Level:       nil,
			Descriptor:  "Comfortable",
			Description: "JUICECON protocols not currently active.",
			AllClear:    true,
		}
	}
}

// LevelDisplay returns the display string for a JUICECON level
func (l Level) LevelDisplay() string {
	if l.AllClear {
		return "ALL CLEAR"
	}
	return "JUICECON " + strconv.Itoa(*l.Level)
}

func intPtr(i int) *int {
	return &i
}
