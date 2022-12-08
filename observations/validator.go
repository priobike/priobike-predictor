package observations

import (
	"fmt"
	"time"
)

// Validate an observation for a given datastream type.
func validateObservation(observation Observation, dsType string) error {
	var shouldValidateTime = false
	switch dsType {
	case "primary_signal":
		shouldValidateTime = true
	case "cycle_second":
		shouldValidateTime = true
	case "detector_car":
		shouldValidateTime = true
	case "detector_bike":
		shouldValidateTime = true
	case "signal_program":
		shouldValidateTime = false
	}
	if shouldValidateTime {
		// Check if the observation is too old.
		timeSince := time.Since(observation.PhenomenonTime)
		if timeSince > 300*time.Second {
			return fmt.Errorf("%s observation is too old: %d seconds", dsType, timeSince/time.Second)
		}
	}
	return nil
}
