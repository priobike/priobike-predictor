package observations

import (
	"testing"
	"time"
)

func TestValidateObservation(t *testing.T) {
	outdatedObservation := Observation{
		PhenomenonTime: time.Unix(0, 0),
	}
	recentObservation := Observation{
		PhenomenonTime: time.Now(), // Unit test should complete quickly enough
	}
	timeSensitiveDsTypes := []string{
		"primary_signal",
		"cycle_second",
		"detector_car",
		"detector_bike",
	}

	for _, dsType := range timeSensitiveDsTypes {
		if validateObservation(outdatedObservation, dsType) == nil {
			t.Errorf("expected validation to error")
			t.FailNow()
		}
		if validateObservation(recentObservation, dsType) != nil {
			t.Errorf("unexpected validation error")
			t.FailNow()
		}
	}
}
