package observations

import (
	"reflect"
	"testing"
	"time"
)

func TestRepo(t *testing.T) {
	mockCycle := &Cycle{
		Outdated: &Observation{
			PhenomenonTime: time.Unix(0, 0),
			// don't care about the result
		},
	}
	primarySignalCycles.Store("1337_1", mockCycle)
	signalProgramCycles.Store("1337_1", mockCycle)
	carDetectorCycles.Store("1337_1", mockCycle)
	bikeDetectorCycles.Store("1337_1", mockCycle)
	cycleSecondCycles.Store("1337_1", mockCycle)
	for _, f := range []func(thingName string) (*Cycle, error){
		GetPrimarySignalCycle,
		GetSignalProgramCycle,
		GetCarDetectorCycle,
		GetBikeDetectorCycle,
		GetCycleSecondCycle,
	} {
		cycle, err := f("1337_1")
		if err != nil {
			t.Fatalf("failure to load current cycle: %s", err.Error())
		}
		if !reflect.DeepEqual(cycle, mockCycle) {
			t.Fatalf("got wrong cycle")
		}
		_, err = f("1337_2") // nonexistent
		if err == nil {
			t.Fatalf("expected error")
		}
	}
	for _, f := range []func(thingName string) (Observation, bool){
		GetCurrentPrimarySignal,
		GetCurrentProgram,
	} {
		observation, ok := f("1337_1")
		if !ok {
			t.Fatalf("failure to load observation")
		}
		if observation.PhenomenonTime != time.Unix(0, 0) {
			t.Fatalf("failure to load correct observation")
		}
		_, ok = f("1337_2") // nonexistent
		if ok {
			t.Fatalf("expected error")
		}
	}
}
