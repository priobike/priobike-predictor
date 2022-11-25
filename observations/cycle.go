package observations

import (
	"fmt"
	"sort"
	"time"
)

// A running cycle of observations.
type Cycle struct {
	// Observations that have been received but not yet completed the full cycle.
	// This slice is *always* sorted by `phenomenonTime`, ascending.
	Pending []Observation
	// The last full (completed) cycle of observations.
	// This slice is *always* sorted by `phenomenonTime`, ascending.
	Completed []Observation
	// The last observation that was received before the start of the current cycle.
	Outdated *Observation
	// The start of the last completed cycle.
	StartTime time.Time
	// The end of the last completed cycle.
	EndTime time.Time
}

// Get the most recent pending observation from a cycle.
func (c Cycle) GetMostRecentObservation() (Observation, error) {
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

// Find an observation for a specific unix second in the cycle.
func (c Cycle) FindObservationForSecond(second int64) (Observation, error) {
	// Check the pending observations in reverse order.
	for i := len(c.Pending) - 1; i >= 0; i-- {
		// If the observation is in the future, skip it.
		if c.Pending[i].PhenomenonTime.Unix() > second {
			continue
		}
		return c.Pending[i], nil
	}
	// Check the completed observations in reverse order.
	for i := len(c.Completed) - 1; i >= 0; i-- {
		// If the observation is in the future, skip it.
		if c.Completed[i].PhenomenonTime.Unix() > second {
			continue
		}
		return c.Completed[i], nil
	}
	// Check the outdated observation.
	if c.Outdated != nil {
		// If the observation is in the future, skip it.
		if c.Outdated.PhenomenonTime.Unix() > second {
			return Observation{}, fmt.Errorf("second out of range")
		}
		return *c.Outdated, nil
	}
	return Observation{}, fmt.Errorf("no observations in cycle")
}

// Find all observations in a time range.
func (c *Cycle) FindObservationsInRange(start, end time.Time) (os []Observation) {
	// Check the pending observations.
	for _, observation := range c.Pending {
		if observation.PhenomenonTime.After(start) && observation.PhenomenonTime.Before(end) {
			os = append(os, observation)
		}
	}
	// Check the completed observations.
	for _, observation := range c.Completed {
		if observation.PhenomenonTime.After(start) && observation.PhenomenonTime.Before(end) {
			os = append(os, observation)
		}
	}
	// Check the outdated observation.
	if c.Outdated != nil {
		if c.Outdated.PhenomenonTime.After(start) && c.Outdated.PhenomenonTime.Before(end) {
			os = append(os, *c.Outdated)
		}
	}
	return os
}

// Add a new observation to a cycle.
func (c *Cycle) add(observation Observation) {
	// If the observation is before the start of the current cycle, it is outdated.
	if observation.PhenomenonTime.Before(c.StartTime) {
		c.Outdated = &observation
		return
	}
	// If the observation is before the end of the current cycle, it is completed.
	if observation.PhenomenonTime.Before(c.EndTime) {
		c.Completed = append(c.Completed, observation)
		sort.Slice(c.Completed, func(i, j int) bool {
			return c.Completed[i].PhenomenonTime.Before(c.Completed[j].PhenomenonTime)
		})
		return
	}
	// Otherwise, the observation is pending.
	c.Pending = append(c.Pending, observation)
	sort.Slice(c.Pending, func(i, j int) bool {
		return c.Pending[i].PhenomenonTime.Before(c.Pending[j].PhenomenonTime)
	})
}

// Complete the observations that have been received in the last cycle.
func (c *Cycle) complete(cycleCompletionTime time.Time) []Observation {
	c.StartTime = c.EndTime
	c.EndTime = cycleCompletionTime
	if len(c.Completed) > 0 {
		c.Outdated = &c.Completed[len(c.Completed)-1]
	} else {
		c.Outdated = nil
	}
	c.Completed = c.Pending
	c.Pending = []Observation{}
	return c.Completed
}

// Pretty print a cycle.
func (c Cycle) String() string {
	return fmt.Sprintf(
		"Pending: %v Completed: %v Outdated: %v StartTime: %v EndTime: %v",
		c.Pending, c.Completed, c.Outdated, c.StartTime, c.EndTime,
	)
}
