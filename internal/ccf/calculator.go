package ccf

import "strconv"

// Level represents a CCF severity level
// Note: CCF goes UP in severity by number as dewpoint goes DOWN
type Level struct {
	Level       *int   `json:"level"`
	Descriptor  string `json:"descriptor"`
	Description string `json:"description"`
	AllClear    bool   `json:"allClear"`
}

// Calculate determines the CCF level from a dewpoint in Fahrenheit
func Calculate(dewpointF float64) Level {
	switch {
	case dewpointF <= 2:
		return Level{
			Level:       intPtr(5),
			Descriptor:  "Walking EMP",
			Description: "I am a walking EMP. Strong aversion to touching doorknobs. Hair is snakes!",
			AllClear:    false,
		}
	case dewpointF <= 8:
		return Level{
			Level:       intPtr(4),
			Descriptor:  "Husk Hands",
			Description: "Hands are husks of their former selves.",
			AllClear:    false,
		}
	case dewpointF <= 14:
		return Level{
			Level:       intPtr(3),
			Descriptor:  "Lotion Failure",
			Description: "This lotion is NOT cutting it.",
			AllClear:    false,
		}
	case dewpointF <= 19:
		return Level{
			Level:       intPtr(2),
			Descriptor:  "Humidifier Check",
			Description: "Do we even have a working humidifier?",
			AllClear:    false,
		}
	case dewpointF <= 25:
		return Level{
			Level:       intPtr(1),
			Descriptor:  "Cotton Mouth",
			Description: "Cotton mouth but no alcohol? Hmmmm. Saline nasal rinse under consideration.",
			AllClear:    false,
		}
	default:
		return Level{
			Level:       nil,
			Descriptor:  "Comfortable",
			Description: "CCF protocols not currently active.",
			AllClear:    true,
		}
	}
}

// LevelDisplay returns the display string for a CCF level
func (l Level) LevelDisplay() string {
	if l.AllClear {
		return "ALL CLEAR"
	}
	return "CCF " + strconv.Itoa(*l.Level)
}

func intPtr(i int) *int {
	return &i
}
