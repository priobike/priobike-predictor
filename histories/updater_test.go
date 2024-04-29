package histories

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/env"
	"predictor/observations"
	"reflect"
	"testing"
	"time"
)

func TestUpdater(t *testing.T) {
	tempDir := t.TempDir()
	env.StaticPath = tempDir

	// Prepare test data
	thingName := "1337_1"
	newCycleStartTime := time.Unix(0, 0)
	newCycleEndTime := time.Unix(90, 0)
	completedPrimarySignalCycle := observations.CycleSnapshot{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(90, 0),
		// Observations that were not yet included in the completed cycle
		Pending: []observations.Observation{
			{
				PhenomenonTime: time.Unix(120, 0), // After the cycle
				Result:         3,                 // Green
			},
		},
		// Observations included in the completed cycle
		Completed: []observations.Observation{
			{
				PhenomenonTime: time.Unix(10, 0),
				Result:         3, // Green
			},
			{
				PhenomenonTime: time.Unix(50, 0),
				Result:         1, // Red
			},
		},
		// Observations before the cycle started
		Outdated: &observations.Observation{
			PhenomenonTime: time.Unix(0, 0),
			Result:         1, // Red
		},
	}
	completedSignalProgramCycle := observations.CycleSnapshot{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(90, 0),
		// Observations that were not yet included in the completed cycle
		Pending: []observations.Observation{
			// No program change after the cycle
		},
		// Observations included in the completed cycle
		Completed: []observations.Observation{
			// No program change during the cycle
		},
		Outdated: &observations.Observation{
			// Last running program before the cycle
			PhenomenonTime: time.Unix(0, 0),
			Result:         10, // Program 10
		},
	}
	completedCycleSecondCycle := observations.CycleSnapshot{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(90, 0),
		// Observations that were not yet included in the completed cycle
		Pending: []observations.Observation{
			{
				PhenomenonTime: time.Unix(90, 0),
			},
		},
		// Observations included in the completed cycle
		Completed: []observations.Observation{
			{
				PhenomenonTime: time.Unix(0, 0),
			},
		},
	}
	completedCarDetectorCycle := observations.CycleSnapshot{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(90, 0),
		Completed: []observations.Observation{
			{
				PhenomenonTime: time.Unix(0, 0),
				Result:         100, // Full occupation of the car detector
			},
		},
	}
	completedBikeDetectorCycle := observations.CycleSnapshot{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(90, 0),
		Completed: []observations.Observation{
			{
				PhenomenonTime: time.Unix(0, 0),
				Result:         100, // Full occupation of the bike detector
			},
		},
	}

	history, err := UpdateHistory(thingName, newCycleStartTime, newCycleEndTime, completedPrimarySignalCycle, completedSignalProgramCycle, completedCycleSecondCycle, completedCarDetectorCycle, completedBikeDetectorCycle)
	if err != nil {
		t.Errorf("error during history update: %s", err.Error())
		t.FailNow()
	}

	expectedPath := fmt.Sprintf("%s/history/1337_1-P10.json", tempDir)
	file, err := os.OpenFile(expectedPath, os.O_RDONLY, 0444)
	if err != nil {
		t.Errorf("could not open history file: %s", err.Error())
		t.FailNow()
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var decodedHistory History
	err = decoder.Decode(&decodedHistory)
	if err != nil {
		t.Errorf("could not decode history: %s", err.Error())
		t.FailNow()
	}
	// Using deep equals in the test is fine
	if !reflect.DeepEqual(history, decodedHistory) {
		t.Errorf("written history does not correspond to returned history: %v != %v", decodedHistory, history)
	}
	if HistoryUpdatesProcessed != 1 {
		t.Errorf("there should be one processed history update")
	}
	if HistoryUpdatesRequested != 1 {
		t.Errorf("there should be one requested history update")
	}
}
