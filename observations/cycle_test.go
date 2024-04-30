package observations

import (
	"testing"
	"time"
)

var testCycle = Cycle{
	Pending: []Observation{
		{
			PhenomenonTime: time.Unix(10, 0),
		},
	},
	Completed: []Observation{
		{
			PhenomenonTime: time.Unix(2, 0),
		},
		{
			PhenomenonTime: time.Unix(1, 0),
		},
	},
	Outdated: &Observation{
		PhenomenonTime: time.Unix(0, 0),
	},
}

func TestGetMostRecentObservation(t *testing.T) {
	snapshot := testCycle.MakeSnapshot()
	o, err := snapshot.GetMostRecentObservation()
	if err != nil {
		t.Fatalf("error getting most recent observation: %s", err.Error())
	}
	if o.PhenomenonTime != time.Unix(10, 0) {
		t.Fatalf("got wrong observation")
	}
}
