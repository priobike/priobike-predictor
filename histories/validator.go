package histories

import (
	"fmt"
	"math"
	"predictor/phases"
	"time"
)

// Check if the phases are valid.
func validatePhases(startTime time.Time, endTime time.Time, events []HistoryPhaseEvent) error {
	// If the phases are empty, they are invalid.
	lenEvents := len(events)
	if lenEvents == 0 {
		return fmt.Errorf("no phases")
	}
	// We need at least one phase before the start time (a "full" cycle).
	if events[0].Time.After(startTime) {
		return fmt.Errorf("no phase before start time")
	}
	// Don't update the history file if a the phases are out of order.
	for i := 1; i < lenEvents; i++ {
		prev := events[i-1]
		curr := events[i]
		// Typical cycles:
		// - Red -> RedAmber -> Green -> Amber -> Red
		// - Red -> Green -> Red
		if prev.Color == phases.Red && curr.Color == phases.Amber {
			// Red -> Amber is disallowed.
			return fmt.Errorf("red -> amber is disallowed")
		}
		if prev.Color == phases.Amber && curr.Color == phases.Green {
			// Amber -> Green is disallowed.
			return fmt.Errorf("amber -> green is disallowed")
		}
		if prev.Color == phases.Amber && curr.Color == phases.RedAmber {
			// Amber -> RedAmber is disallowed.
			return fmt.Errorf("amber -> redamber is disallowed")
		}
		if prev.Color == phases.Green && curr.Color == phases.RedAmber {
			// Green -> RedAmber is disallowed.
			return fmt.Errorf("green -> redamber is disallowed")
		}
		if prev.Color == phases.RedAmber && curr.Color == phases.Red {
			// RedAmber -> Red is disallowed.
			return fmt.Errorf("redamber -> red is disallowed")
		}
		if prev.Color == phases.RedAmber && curr.Color == phases.Amber {
			// RedAmber -> Amber is disallowed.
			return fmt.Errorf("redamber -> amber is disallowed")
		}

		// Validate the length of the phase.
		var phaseDuration int
		if i == lenEvents-1 {
			phaseDuration = int(math.Abs(endTime.Sub(curr.Time).Seconds()))
		} else {
			phaseDuration = int(math.Abs(events[i+1].Time.Sub(curr.Time).Seconds()))
		}
		// RedAmber phases after a red phase should not be longer than 10 seconds.
		if prev.Color == phases.Red && curr.Color == phases.RedAmber && phaseDuration > 10 {
			return fmt.Errorf("redamber phase is too long")
		}
		// Amber phases after a green phase should not be longer than 10 seconds.
		if prev.Color == phases.Green && curr.Color == phases.Amber && phaseDuration > 10 {
			return fmt.Errorf("amber phase is too long")
		}
	}
	return nil
}
