package observations

import "time"

// Run a cleanup on the observations.
func cleanup() {
	// Truncate all cycles to the maximum length, to avoid storing too many observations.
	primarySignalCycles.Range(func(key, value interface{}) bool {
		cycle := value.(*Cycle)
		cycle.truncatePending(20)
		return true
	})
	signalProgramCycles.Range(func(key, value interface{}) bool {
		cycle := value.(*Cycle)
		cycle.truncatePending(5)
		return true
	})
	carDetectorCycles.Range(func(key, value interface{}) bool {
		cycle := value.(*Cycle)
		cycle.truncatePending(300)
		return true
	})
	bikeDetectorCycles.Range(func(key, value interface{}) bool {
		cycle := value.(*Cycle)
		cycle.truncatePending(300)
		return true
	})
	cycleSecondCycles.Range(func(key, value interface{}) bool {
		cycle := value.(*Cycle)
		cycle.truncatePending(5)
		return true
	})
}

// Run a periodic cleanup of the observations.
func RunCleanupPeriodically() {
	for {
		cleanup()
		time.Sleep(60 * time.Second)
	}
}
