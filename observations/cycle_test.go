package observations

import (
	"testing"
	"time"
)

var generateTestCycle = func() Cycle {
	return Cycle{
		StartTime: time.Unix(2, 0),
		EndTime:   time.Unix(5, 0),
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
				PhenomenonTime: time.Unix(3, 0),
			},
		},
		Outdated: &Observation{
			PhenomenonTime: time.Unix(1, 0),
		},
	}
}

func TestGetMostRecentObservation(t *testing.T) {
	testCycle := generateTestCycle()
	snapshot := testCycle.MakeSnapshot()
	o, err := snapshot.GetMostRecentObservation()
	if err != nil {
		t.Fatalf("error getting most recent observation: %s", err.Error())
	}
	if o.PhenomenonTime != time.Unix(10, 0) {
		t.Fatalf("got wrong observation")
	}
}

func TestTruncatePending(t *testing.T) {
	testCycle := generateTestCycle()
	testCycle.truncatePending(0)
	if len(testCycle.Pending) != 0 {
		t.Fatalf("pending part of cycle should no longer contain observations")
	}
}

func TestAddObservation(t *testing.T) {
	testCycle := generateTestCycle()
	testCycle.add(Observation{
		PhenomenonTime: time.Unix(0, 0),
	})
	if testCycle.Outdated == nil || testCycle.Outdated.PhenomenonTime != time.Unix(0, 0) {
		t.Fatalf("should add observation as outdated")
	}

	testCycle = generateTestCycle()
	testCycle.add(Observation{
		PhenomenonTime: time.Unix(4, 0),
	})
	if len(testCycle.Completed) != 3 {
		t.Fatalf("expected observation to be added in completed")
	}
	if testCycle.Completed[2].PhenomenonTime != time.Unix(4, 0) {
		t.Fatalf("added observation in wrong spot")
	}

	testCycle = generateTestCycle()
	testCycle.add(Observation{
		PhenomenonTime: time.Unix(15, 0),
	})
	if len(testCycle.Pending) != 2 {
		t.Fatalf("expected observation to be added to pending")
	}
	if testCycle.Pending[1].PhenomenonTime != time.Unix(15, 0) {
		t.Fatalf("added observation in wrong spot")
	}
}

func TestCompleteCycle(t *testing.T) {
	testCycle := generateTestCycle()
	snapshot, err := testCycle.complete(time.Unix(5, 0), time.Unix(15, 0))
	if err != nil {
		t.Fatalf("error during completion of test cycle: %s", err)
	}
	if snapshot.Outdated == nil || snapshot.Outdated.PhenomenonTime != time.Unix(3, 0) {
		t.Fatalf("did not move out completed observations to outdated correctly")
	}
	if len(snapshot.Completed) != 1 {
		t.Fatalf("did not move pending observation to completed correctly")
	}
	if snapshot.Completed[0].PhenomenonTime != time.Unix(10, 0) {
		t.Fatalf("did not move pending observation to completed correctly")
	}
	if len(snapshot.Pending) != 0 {
		t.Fatalf("did not cleanup pending observations correctly")
	}
}
