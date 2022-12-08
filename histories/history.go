package histories

import (
	"predictor/util"
	"time"
)

// A phase change of a traffic light.
// This is a simplified, less generic version of the raw observation.
type HistoryPhaseEvent struct {
	// The time when the observation was made.
	Time time.Time `json:"time"`
	// The color of the signal.
	Color byte `json:"color"`
}

// A detection of a vehicle, either a car or a bike.
// The event contains the name of the signal that detected the vehicle.
type HistoryDetectionEvent struct {
	// The time when the observation was made.
	Time time.Time `json:"time"`
	// The traffic light that detected the vehicle.
	Signal string `json:"signal"`
	// The percentage of the detector.
	Pct byte `json:"pct"`
}

// A completed cycle in the history of a traffic light.
type HistoryCycle struct {
	// The time when the cycle started.
	StartTime time.Time `json:"startTime"`
	// The time when the cycle ended.
	EndTime time.Time `json:"endTime"`
	// The program that was currently running during the cycle.
	Program *byte `json:"program"`
	// The signal phase during or before the cycle, sorted by time.
	Phases []HistoryPhaseEvent `json:"phases"`
	// The detected cars during or before the cycle, sorted by time.
	Cars []HistoryDetectionEvent `json:"cars"`
	// The detected bikes during or before the cycle, sorted by time.
	Bikes []HistoryDetectionEvent `json:"bikes"`
}

type History struct {
	// The rows (cycles) of the history.
	Cycles []HistoryCycle `json:"cycles"`
}

// Flatten a history into an array of cycles of seconds by their color.
func (h History) Flatten() [][]byte {
	if len(h.Cycles) == 0 {
		return [][]byte{}
	}
	flattenedCycles := [][]byte{}
	for _, cycle := range h.Cycles {
		if len(cycle.Phases) == 0 {
			continue
		}
		startTime := cycle.StartTime.Unix()
		endTime := cycle.EndTime.Unix()
		if endTime < startTime {
			continue
		}
		if endTime-startTime < 10 {
			continue // Avoid too short cycles.
		}
		if endTime-startTime > 300 {
			continue // Avoid too long cycles.
		}
		flattenedCycle := make([]byte, endTime-startTime)
		for i := 0; i < len(cycle.Phases); i++ {
			startIdx := util.Max64(cycle.Phases[i].Time.Unix()-startTime, 0)
			var endIdx int64
			if i == len(cycle.Phases)-1 {
				// Fill the rest of the cycle with the last phase.
				endIdx = util.Max64(startIdx, endTime-startTime)
			} else {
				// Fill until the next phase.
				endIdx = util.Max64(startIdx, util.Min64(cycle.Phases[i+1].Time.Unix()-startTime, endTime-startTime))
			}
			for j := startIdx; j < endIdx; j++ {
				flattenedCycle[j] = cycle.Phases[i].Color
			}
		}
		if len(flattenedCycle) > 0 {
			flattenedCycles = append(flattenedCycles, flattenedCycle)
		}
	}
	return flattenedCycles
}
