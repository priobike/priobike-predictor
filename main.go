package main

import (
	"predictor/history"
	"predictor/observations"
	"predictor/things"
)

func main() {
	go things.Sync()                        // Periodically syncs the things from the SensorThings API.
	onCycleFinished := history.Build        // Builds the history on each cycle.
	go observations.Listen(onCycleFinished) // Listens to the MQTT broker for observations.
	go history.UpdateIndex()                // Periodically updates the history index file.

	// Wait forever.
	select {}
}
