package observations

import (
	"fmt"
	"sync"
)

// A map that contains all `primary_signal` cycles to the Thing name.
// Primary signal observations tell which "color" the traffic light is currently showing.
var primarySignalCycles = &sync.Map{}

func GetPrimarySignalCycle(thingName string) (*Cycle, error) {
	cycle, ok := primarySignalCycles.Load(thingName)
	if !ok {
		return nil, fmt.Errorf("no cycle found for thing %s", thingName)
	}
	return cycle.(*Cycle), nil
}

// Get the current color for a given thing.
func GetCurrentPrimarySignal(thingName string) (Observation, bool) {
	cycle, err := GetPrimarySignalCycle(thingName)
	if err != nil {
		return Observation{}, false
	}
	snapshot := cycle.MakeSnapshot()
	currentState, err := snapshot.GetMostRecentObservation()
	if err != nil {
		return Observation{}, false
	}
	return currentState, true
}

// A map that contains all received `signal_program` observations to the Thing name.
// Signal program observations tell which program the traffic light is currently running.
var signalProgramCycles = &sync.Map{}

func GetSignalProgramCycle(thingName string) (*Cycle, error) {
	cycle, ok := signalProgramCycles.Load(thingName)
	if !ok {
		return nil, fmt.Errorf("no cycle found for thing %s", thingName)
	}
	return cycle.(*Cycle), nil
}

// Get the currently running program for a given thing.
func GetCurrentProgram(thingName string) (Observation, bool) {
	cycle, err := GetSignalProgramCycle(thingName)
	if err != nil {
		return Observation{}, false
	}
	snapshot := cycle.MakeSnapshot()
	currentState, err := snapshot.GetMostRecentObservation()
	if err != nil {
		return Observation{}, false
	}
	return currentState, true
}

// A map that contains all received `detector_car` observations to the Thing name.
// Detector car observations tell when a car is detected, from 0 to 100 pct.
var carDetectorCycles = &sync.Map{}

func GetCarDetectorCycle(thingName string) (*Cycle, error) {
	cycle, ok := carDetectorCycles.Load(thingName)
	if !ok {
		return nil, fmt.Errorf("no cycle found for thing %s", thingName)
	}
	return cycle.(*Cycle), nil
}

// A map that contains all received `detector_bike` observations to the Thing name.
// Detector bike observations tell when a bike is detected, from 0 to 100 pct.
var bikeDetectorCycles = &sync.Map{}

func GetBikeDetectorCycle(thingName string) (*Cycle, error) {
	cycle, ok := bikeDetectorCycles.Load(thingName)
	if !ok {
		return nil, fmt.Errorf("no cycle found for thing %s", thingName)
	}
	return cycle.(*Cycle), nil
}

// A map that contains all received `cycle_second` observations to the Thing name.
// Cycle second observations tell when a new cycle starts.
var cycleSecondCycles = &sync.Map{}

func GetCycleSecondCycle(thingName string) (*Cycle, error) {
	cycle, ok := cycleSecondCycles.Load(thingName)
	if !ok {
		return nil, fmt.Errorf("no cycle found for thing %s", thingName)
	}
	return cycle.(*Cycle), nil
}
