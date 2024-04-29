package histories

import (
	"predictor/phases"
	"strings"
	"testing"
	"time"
)

func TestValidateEmptyPhases(t *testing.T) {
	err := validatePhases(
		time.Unix(0, 0),
		time.Unix(90, 0),
		[]HistoryPhaseEvent{},
	)
	if err == nil || err.Error() != "no phases" {
		t.Errorf("unexpected validation result: %v", err)
		t.FailNow()
	}
}

func TestValidateNoPhaseBefore(t *testing.T) {
	err := validatePhases(
		time.Unix(0, 0),
		time.Unix(90, 0),
		[]HistoryPhaseEvent{
			{
				Time:  time.Unix(10, 0),
				Color: phases.Green, // Green
			},
		},
	)
	if err == nil || err.Error() != "no phase before start time" {
		t.Errorf("unexpected validation result: %v", err)
		t.FailNow()
	}
}

func TestDisallowedOrders(t *testing.T) {
	disallowedOrders := [][]HistoryPhaseEvent{
		{
			{
				Time:  time.Unix(0, 0),
				Color: phases.Red,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.Amber,
			},
		},
		{
			{
				Time:  time.Unix(0, 0),
				Color: phases.Amber,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.Green,
			},
		},
		{
			{
				Time:  time.Unix(0, 0),
				Color: phases.Amber,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.RedAmber,
			},
		},
		{
			{
				Time:  time.Unix(0, 0),
				Color: phases.Green,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.RedAmber,
			},
		},
		{
			{
				Time:  time.Unix(0, 0),
				Color: phases.RedAmber,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.Red,
			},
		},
		{
			{
				Time:  time.Unix(0, 0),
				Color: phases.RedAmber,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.Amber,
			},
		},
	}

	for _, order := range disallowedOrders {
		err := validatePhases(time.Unix(0, 0), time.Unix(10, 0), order)
		// Must throw an error
		if err == nil {
			t.Errorf("erroneous phase order was not detected")
			t.FailNow()
		}
		if !strings.HasSuffix(err.Error(), "is disallowed") {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	}
}

func TestValidatePhaseDuration(t *testing.T) {
	err := validatePhases(
		time.Unix(0, 0),
		time.Unix(90, 0),
		[]HistoryPhaseEvent{
			{
				Time:  time.Unix(0, 0),
				Color: phases.Red,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.RedAmber,
			},
		},
	)
	if err == nil || err.Error() != "redamber phase is too long" {
		t.Errorf("unexpected validation result: %v", err)
		t.FailNow()
	}

	err = validatePhases(
		time.Unix(0, 0),
		time.Unix(90, 0),
		[]HistoryPhaseEvent{
			{
				Time:  time.Unix(0, 0),
				Color: phases.Green,
			},
			{
				Time:  time.Unix(10, 0),
				Color: phases.Amber,
			},
		},
	)
	if err == nil || err.Error() != "amber phase is too long" {
		t.Errorf("unexpected validation result: %v", err)
		t.FailNow()
	}
}
