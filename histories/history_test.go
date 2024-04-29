package histories

import (
	"reflect"
	"testing"
	"time"
)

func TestFlattenHistory(t *testing.T) {
	exampleCycle := HistoryCycle{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(10, 0), // Usually cycles are longer, but suffices for testing
		Phases: []HistoryPhaseEvent{
			// Green at 0 seconds
			{
				Time:  time.Unix(0, 0),
				Color: 3,
			},
			// Red at 5 seconds
			{
				Time:  time.Unix(5, 0),
				Color: 1,
			},
		},
	}
	history := History{
		Cycles: []HistoryCycle{
			exampleCycle,
			exampleCycle,
		},
	}
	expectedArr := [][]byte{
		{
			3, 3, 3, 3, 3,
			1, 1, 1, 1, 1,
		},
		{
			3, 3, 3, 3, 3,
			1, 1, 1, 1, 1,
		},
	}
	actualArr := history.Flatten()
	if !reflect.DeepEqual(expectedArr, actualArr) {
		t.Errorf("unexpected result")
	}
}

func TestErroneousCyclesArePruned(t *testing.T) {
	validCycle := HistoryCycle{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(10, 0), // Usually cycles are longer, but suffices for testing
		Phases: []HistoryPhaseEvent{
			// Green at 0 seconds
			{
				Time:  time.Unix(0, 0),
				Color: 3,
			},
			// Red at 5 seconds
			{
				Time:  time.Unix(5, 0),
				Color: 1,
			},
		},
	}
	startTimeBeforeEndTime := HistoryCycle{
		StartTime: time.Unix(10, 0),
		EndTime:   time.Unix(0, 0),
		Phases: []HistoryPhaseEvent{
			// Green at 0 seconds
			{
				Time:  time.Unix(0, 0),
				Color: 3,
			},
			// Red at 5 seconds
			{
				Time:  time.Unix(5, 0),
				Color: 1,
			},
		},
	}
	tooShort := HistoryCycle{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(5, 0),
		Phases: []HistoryPhaseEvent{
			// Green at 0 seconds
			{
				Time:  time.Unix(0, 0),
				Color: 3,
			},
			// Red at 3 seconds
			{
				Time:  time.Unix(3, 0),
				Color: 1,
			},
		},
	}
	tooLong := HistoryCycle{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(400, 0),
		Phases: []HistoryPhaseEvent{
			// Green at 0 seconds
			{
				Time:  time.Unix(0, 0),
				Color: 3,
			},
			// Red at 100 seconds
			{
				Time:  time.Unix(100, 0),
				Color: 1,
			},
		},
	}
	empty := HistoryCycle{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(90, 0),
	}
	history := History{
		Cycles: []HistoryCycle{
			validCycle,
			startTimeBeforeEndTime,
			tooShort,
			tooLong,
			empty,
		},
	}
	expectedArr := [][]byte{
		{
			3, 3, 3, 3, 3,
			1, 1, 1, 1, 1,
		},
	}
	actualArr := history.Flatten()
	// Using DeepEqual is fine for testing
	if !reflect.DeepEqual(expectedArr, actualArr) {
		t.Errorf("unexpected result")
	}
}

func TestEmptyHistoryFlattening(t *testing.T) {
	h := History{}
	// Using DeepEqual is fine for testing
	if !reflect.DeepEqual(h.Flatten(), [][]byte{}) {
		t.Errorf("empty history flatten should result in empty 2d slice")
	}
}
