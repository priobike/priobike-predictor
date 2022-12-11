package observations

import "time"

// A callback that is called when a `primary_signal` message is received.
var PrimarySignalCallback = func(thingName string) {}

// A callback that is called when a `signal_program` message is received.
var SignalProgramCallback = func(thingName string) {}

// A callback that is called when a `detector_car` message is received.
var CarDetectorCallback = func(thingName string) {}

// A callback that is called when a `detector_bike` message is received.
var BikeDetectorCallback = func(thingName string) {}

// A callback that is called when a `cycle_second` message is received.
var CycleSecondCallback = func(
	thingName string,
	newCycleStartTime time.Time, newCycleEndTime time.Time,
	completedPrimarySignalCycle CycleSnapshot,
	completedSignalProgramCycle CycleSnapshot,
	completedCycleSecondCycle CycleSnapshot,
	completedCarDetectorCycle CycleSnapshot,
	completedBikeDetectorCycle CycleSnapshot,
) {
	// The default implementation does nothing.
}
