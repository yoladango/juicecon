package index

import (
	"juicecon-golang/internal/ccf"
	"juicecon-golang/internal/juicecon"
)

// SystemID identifies which sub-system is active
type SystemID string

const (
	SystemJuiceCon SystemID = "juicecon"
	SystemCCF      SystemID = "ccf"
	SystemNone     SystemID = "none"
)

// Result represents the unified assessment from the parent system
type Result struct {
	ActiveSystem SystemID `json:"activeSystem"`
	SystemName   string   `json:"systemName"`
	Level        *int     `json:"level"`
	LevelDisplay string   `json:"levelDisplay"`
	Descriptor   string   `json:"descriptor"`
	Description  string   `json:"description"`
	AllClear     bool     `json:"allClear"`
}

// Evaluate takes a dewpoint in Fahrenheit and returns the unified result,
// activating whichever sub-system (JuiceCon or CCF) applies.
//
// Dewpoint >= 60°F: JuiceCon territory (humidity)
// Dewpoint 26-59°F: Comfort zone (all clear)
// Dewpoint <= 25°F: CCF territory (dryness)
func Evaluate(dewpointF float64) Result {
	if dewpointF >= 60 {
		jc := juicecon.Calculate(dewpointF)
		return Result{
			ActiveSystem: SystemJuiceCon,
			SystemName:   "JUICECON",
			Level:        jc.Level,
			LevelDisplay: jc.LevelDisplay(),
			Descriptor:   jc.Descriptor,
			Description:  jc.Description,
			AllClear:     jc.AllClear,
		}
	}

	if dewpointF <= 25 {
		cf := ccf.Calculate(dewpointF)
		return Result{
			ActiveSystem: SystemCCF,
			SystemName:   "CCF",
			Level:        cf.Level,
			LevelDisplay: cf.LevelDisplay(),
			Descriptor:   cf.Descriptor,
			Description:  cf.Description,
			AllClear:     cf.AllClear,
		}
	}

	// Comfort zone: neither system active
	return Result{
		ActiveSystem: SystemNone,
		SystemName:   "",
		Level:        nil,
		LevelDisplay: "ALL CLEAR",
		Descriptor:   "Comfortable",
		Description:  "All atmospheric moisture protocols inactive. Conditions nominal.",
		AllClear:     true,
	}
}
