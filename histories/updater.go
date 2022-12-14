package histories

import (
	"fmt"
	"predictor/env"
	"predictor/log"
	"predictor/observations"
	"sort"
	"sync/atomic"
	"time"
)

var requested uint64 = 0
var canceled uint64 = 0
var processed uint64 = 0

// Update the history file for the given thing.
func UpdateHistory(
	thingName string,
	newCycleStartTime time.Time,
	newCycleEndTime time.Time,
	completedPrimarySignalCycle observations.CycleSnapshot,
	completedSignalProgramCycle observations.CycleSnapshot,
	completedCycleSecondCycle observations.CycleSnapshot,
	completedCarDetectorCycle observations.CycleSnapshot,
	completedBikeDetectorCycle observations.CycleSnapshot,
) (History, error) {
	if (requested%1000) == 0 && requested > 0 {
		log.Info.Printf("History file updates requested %d, processed %d, canceled %d", requested, processed, canceled)
	}

	atomic.AddUint64(&requested, 1)

	// Reconstruct the signal values in the cycle, from StartTime to EndTime.
	phases := []HistoryPhaseEvent{}
	phasesTimes := make(map[int64]byte) // For O(1) duplicate detection.
	if completedPrimarySignalCycle.Outdated != nil {
		event := HistoryPhaseEvent{
			Time:  completedPrimarySignalCycle.Outdated.PhenomenonTime,
			Color: completedPrimarySignalCycle.Outdated.Result,
		}
		phases = append(phases, event)
		phasesTimes[event.Time.Unix()] = event.Color
	}
	for _, o := range completedPrimarySignalCycle.Completed {
		event := HistoryPhaseEvent{
			Time:  o.PhenomenonTime,
			Color: o.Result,
		}
		// Check if we have a duplicate event.
		if color, ok := phasesTimes[event.Time.Unix()]; ok && color == event.Color {
			continue // Skip duplicate events.
		}
		phases = append(phases, event)
		phasesTimes[event.Time.Unix()] = event.Color
	}
	// Sort the phases by time.
	sort.Slice(phases, func(i, j int) bool {
		return phases[i].Time.Before(phases[j].Time)
	})
	err := validatePhases(newCycleStartTime, newCycleEndTime, phases)
	if err != nil {
		atomic.AddUint64(&canceled, 1)
		return History{}, fmt.Errorf("phase validity check failed: %v", err)
	}

	// Check the car detector state for the thing.
	var cars = []HistoryDetectionEvent{}
	carsTimes := make(map[int64]string) // For O(1) duplicate detection.
	// Check if there was any car detected before this cycle.
	if completedCarDetectorCycle.Outdated != nil {
		event := HistoryDetectionEvent{
			Time:   completedCarDetectorCycle.Outdated.PhenomenonTime,
			Signal: thingName,
			Pct:    completedCarDetectorCycle.Outdated.Result,
		}
		cars = append(cars, event)
		carsTimes[event.Time.Unix()] = event.Signal
	}
	// Check if there was any car detected within this cycle.
	for _, oWithin := range completedCarDetectorCycle.Completed {
		event := HistoryDetectionEvent{
			Time:   oWithin.PhenomenonTime,
			Signal: thingName,
			Pct:    oWithin.Result,
		}
		// Check if we have a duplicate event.
		if signal, ok := carsTimes[event.Time.Unix()]; ok && signal == event.Signal {
			continue // Skip duplicate events.
		}
		cars = append(cars, event)
		carsTimes[event.Time.Unix()] = event.Signal
	}
	sort.Slice(cars, func(i, j int) bool {
		return cars[i].Time.Before(cars[j].Time)
	})

	// Check the bike detector state for the thing.
	var bikes = []HistoryDetectionEvent{}
	bikesTimes := make(map[int64]string) // For O(1) duplicate detection.
	// Check if there was any bike detected before this cycle.
	if completedBikeDetectorCycle.Outdated != nil {
		event := HistoryDetectionEvent{
			Time:   completedBikeDetectorCycle.Outdated.PhenomenonTime,
			Signal: thingName,
			Pct:    completedBikeDetectorCycle.Outdated.Result,
		}
		bikes = append(bikes, event)
		bikesTimes[event.Time.Unix()] = event.Signal
	}
	// Check if there was any bike detected within this cycle.
	for _, oWithin := range completedBikeDetectorCycle.Completed {
		event := HistoryDetectionEvent{
			Time:   oWithin.PhenomenonTime,
			Signal: thingName,
			Pct:    oWithin.Result,
		}
		// Check if we have a duplicate event.
		if signal, ok := bikesTimes[event.Time.Unix()]; ok && signal == event.Signal {
			continue // Skip duplicate events.
		}
		bikes = append(bikes, event)
		bikesTimes[event.Time.Unix()] = event.Signal
	}
	sort.Slice(bikes, func(i, j int) bool {
		return bikes[i].Time.Before(bikes[j].Time)
	})

	historyCycle := &HistoryCycle{
		StartTime: newCycleStartTime,
		EndTime:   newCycleEndTime,
		Phases:    phases,
		Cars:      cars,
		Bikes:     bikes,
	}

	// Append this history to the history file.
	var path = fmt.Sprintf("%s/history/%s.json", env.StaticPath, thingName)
	// Get the last program that was running on the signal.
	programObservation, err := completedSignalProgramCycle.GetMostRecentObservation()
	if err == nil {
		programId := programObservation.Result
		path = fmt.Sprintf("%s/history/%s-P%d.json", env.StaticPath, thingName, programId)
		historyCycle.Program = &programId
	}

	history, err := appendToHistoryFile(path, *historyCycle)
	if err != nil {
		atomic.AddUint64(&canceled, 1)
		return History{}, err
	}

	atomic.AddUint64(&processed, 1)
	return history, nil
}
