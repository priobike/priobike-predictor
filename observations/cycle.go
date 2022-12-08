package observations

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// A running cycle of observations.
// A cycle is ended when a new `cycle_time` observation is arrived.
// When this observation arrives, the pending observations are moved to the completed
// observations, and the most recent completed observation is orphaned as outdated.
// In this way we will always have at least one last observation.
type Cycle struct {
	// Observations that have been received but not yet completed the full cycle.
	// This slice is *always* sorted by `phenomenonTime`, ascending.
	Pending []Observation
	// A RW lock is used to protect the pending observations.
	PendingLock sync.RWMutex
	// The last full (completed) cycle of observations, after a `cycle_time` observation
	// was received. This slice is *always* sorted by `phenomenonTime`, ascending.
	Completed []Observation
	// A RW lock is used to protect the completed observations.
	CompletedLock sync.RWMutex
	// The last observation that was received before the start of the current cycle.
	Outdated *Observation
	// A RW lock is used to protect the outdated observation.
	OutdatedLock sync.RWMutex
	// The start of the last completed cycle. This is marked by
	// the time when we received the corresponding `cycle_time` observation.
	StartTime time.Time
	// A RW lock is used to protect the start time.
	StartTimeLock sync.RWMutex
	// The end of the last completed cycle. This is marked by
	// the time when we received the corresponding `cycle_time` observation.
	EndTime time.Time
	// A RW lock is used to protect the end time.
	EndTimeLock sync.RWMutex
}

// A snapshot of a cycle.
type CompletedCycle struct {
	// The start of the cycle.
	StartTime time.Time
	// The end of the cycle.
	EndTime time.Time
	// The completed observations in the cycle.
	Completed []Observation
	// The outdated observation in the cycle.
	Outdated *Observation
}

// Lock all locks in the cycle.
func (c *Cycle) Lock() {
	c.PendingLock.Lock()
	c.CompletedLock.Lock()
	c.OutdatedLock.Lock()
	c.StartTimeLock.Lock()
	c.EndTimeLock.Lock()
}

// Unlock all locks in the cycle.
func (c *Cycle) Unlock() {
	c.PendingLock.Unlock()
	c.CompletedLock.Unlock()
	c.OutdatedLock.Unlock()
	c.StartTimeLock.Unlock()
	c.EndTimeLock.Unlock()
}

// Lock all locks in the cycle, read-only.
func (c *Cycle) RLock() {
	c.PendingLock.RLock()
	c.CompletedLock.RLock()
	c.OutdatedLock.RLock()
	c.StartTimeLock.RLock()
	c.EndTimeLock.RLock()
}

// Unlock all locks in the cycle, read-only.
func (c *Cycle) RUnlock() {
	c.PendingLock.RUnlock()
	c.CompletedLock.RUnlock()
	c.OutdatedLock.RUnlock()
	c.StartTimeLock.RUnlock()
	c.EndTimeLock.RUnlock()
}

func (c *Cycle) GetPending() []Observation {
	c.RLock()
	defer c.RUnlock()
	return c.Pending
}

// Truncate the pending observations to the given length.
// This will remove the oldest observations.
func (c *Cycle) TruncatePending(length int) {
	c.Lock()
	defer c.Unlock()
	if length > len(c.Pending) {
		return
	}
	c.Pending = c.Pending[len(c.Pending)-length:]
}

// Get the most recent pending observation from a cycle.
// This will traverse the pending, completed and outdated observations.
func (c *Cycle) GetMostRecentObservation() (Observation, error) {
	c.RLock()
	defer c.RUnlock()
	// Check the pending observations.
	if len(c.Pending) == 0 {
		// Check if there are any completed observations.
		if len(c.Completed) == 0 {
			// Check if there is an outdated observation.
			if c.Outdated == nil {
				return Observation{}, fmt.Errorf("no observations in cycle")
			}
			return *c.Outdated, nil
		}
		return c.Completed[len(c.Completed)-1], nil
	}
	return c.Pending[len(c.Pending)-1], nil
}

// Get the most recent pending observation from a cycle.
func (c CompletedCycle) GetMostRecentObservation() (Observation, error) {
	// Check if there are any completed observations.
	if len(c.Completed) == 0 {
		// Check if there is an outdated observation.
		if c.Outdated == nil {
			return Observation{}, fmt.Errorf("no observations in cycle")
		}
		return *c.Outdated, nil
	}
	return c.Completed[len(c.Completed)-1], nil
}

// Add a new observation to a cycle.
// This method should be called when a new observation arrives via MQTT.
// Running this function will ensure that the observations are correctly sorted.
// The observation should always be added to the pending observations.
// However if the observations arrive out of order, this will add the observation
// to the completed or outdated observations.
func (c *Cycle) add(observation Observation) {
	c.Lock()
	defer c.Unlock()
	// If the observation is before the start of the current cycle, it is outdated.
	if observation.PhenomenonTime.Before(c.StartTime) {
		c.Outdated = &observation
		return
	}
	// If the observation is before the end of the current cycle, it is completed.
	if observation.PhenomenonTime.Before(c.EndTime) {
		c.Completed = append(c.Completed, observation)
		if len(c.Completed) > 1 && observation.PhenomenonTime.Before(c.Completed[len(c.Completed)-2].PhenomenonTime) {
			sort.Slice(c.Completed, func(i, j int) bool {
				return c.Completed[i].PhenomenonTime.Before(c.Completed[j].PhenomenonTime)
			})
		}
		return
	}
	// Otherwise, the observation is pending.
	c.Pending = append(c.Pending, observation)
	if len(c.Pending) > 1 && observation.PhenomenonTime.Before(c.Pending[len(c.Pending)-2].PhenomenonTime) {
		sort.Slice(c.Pending, func(i, j int) bool {
			return c.Pending[i].PhenomenonTime.Before(c.Pending[j].PhenomenonTime)
		})
	}
}

// Complete the observations that have been received in the last cycle.
// This will perform the following actions:
// 1. Move the start time and end time of the cycle "to the right" on the time scale.
// 2. Move all pending observations to completed, but only if they are within the new time frame.
// 3. Keep the most recent completed observation as outdated.
// Returns a snapshot copy of the cycle that was completed.
func (c *Cycle) complete(cycleStartTime time.Time, cycleEndTime time.Time) (CompletedCycle, error) {
	c.Lock()
	defer c.Unlock()

	// Make some sanity checks.
	if cycleEndTime.Before(cycleStartTime) {
		return CompletedCycle{}, fmt.Errorf("cycle completion time is before end time")
	}

	c.StartTime = cycleStartTime
	c.EndTime = cycleEndTime

	// Wait until we have a full cycle (meaning start and end time are set).
	if c.StartTime.IsZero() || c.EndTime.IsZero() {
		return CompletedCycle{}, fmt.Errorf("cycle not yet complete")
	}

	// Collect all observations in the Cycle, in ascending order.
	allObservations := []Observation{}
	if c.Outdated != nil {
		allObservations = append(allObservations, *c.Outdated)
	}
	allObservations = append(allObservations, c.Completed...)
	allObservations = append(allObservations, c.Pending...)

	// Make sure that the observations are sorted by `phenomenonTime`.
	sort.Slice(allObservations, func(i, j int) bool {
		return allObservations[i].PhenomenonTime.Before(allObservations[j].PhenomenonTime)
	})

	// Reset the observations.
	c.Outdated = nil
	c.Completed = []Observation{}
	c.Pending = []Observation{}

	// Note that the observations are sorted by `phenomenonTime`` ascending.
	for i, observation := range allObservations {
		if observation.PhenomenonTime.Before(c.StartTime) {
			// If the observation is before the start time, mark it as outdated.
			c.Outdated = &allObservations[i] // Don't use the observation variable!
		} else if observation.PhenomenonTime.Before(c.EndTime) {
			// If the observation is before the end time, mark it as completed.
			c.Completed = append(c.Completed, observation)
		} else {
			// If the observation is after the end time, mark it as pending.
			c.Pending = append(c.Pending, observation)
		}
	}

	return CompletedCycle{
		StartTime: c.StartTime,
		EndTime:   c.EndTime,
		Outdated:  c.Outdated,
		Completed: c.Completed,
	}, nil
}
