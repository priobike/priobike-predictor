package main

import (
	"predictor/histories"
	"predictor/monitor"
	"predictor/observations"
	"predictor/predictions"
	"predictor/things"
	"time"
)

func main() {
	// Sync the things.
	things.SyncThings()
	// Update the history index once for the cycle visualizer.
	histories.UpdateHistoryIndex()
	// Update the history index periodically for the cycle visualizer.
	go histories.UpdateHistoryIndexPeriodically()
	// Prefetch all most recent observations.
	observations.PrefetchMostRecentObservations()
	// Connect to the mqtt broker and listen for observations.
	observations.ConnectObservationListener()
	// Check periodically how many messages were received.
	go observations.CheckReceivedMessagesPeriodically()
	// Run a cleanup periodically.
	go observations.RunCleanupPeriodically()
	// Connect the prediction publisher.
	predictions.ConnectMQTTClient()
	// Publish all predictions.
	predictions.PublishAllBestPredictions()
	// Publish all predictions periodically.
	go predictions.PublishAllBestPredictionsPeriodically()
	// Listen for predictions on the old prediction broker.
	monitor.ListenForOldPredictions()
	// Update the prediction comparison once for the dashboard.
	monitor.UpdateMetricsFile()
	// Update the prediction comparison periodically for the dashboard.
	go monitor.UpdateMetricsFilePeriodically()
	// Bind the callbacks.
	observations.PrimarySignalCallback = func(thingName string) {
		predictions.PublishBestPrediction(thingName)
	}
	observations.SignalProgramCallback = func(thingName string) {
		predictions.PublishBestPrediction(thingName)
	}
	observations.CarDetectorCallback = func(thingName string) {
		// This currently has no influence on the predictions.
	}
	observations.BikeDetectorCallback = func(thingName string) {
		// This currently has no influence on the predictions.
	}
	observations.CycleSecondCallback = func(
		thingName string,
		newCycleStartTime time.Time, newCycleEndTime time.Time,
		completedPrimarySignalCycle observations.CycleSnapshot,
		completedSignalProgramCycle observations.CycleSnapshot,
		completedCycleSecondCycle observations.CycleSnapshot,
		completedCarDetectorCycle observations.CycleSnapshot,
		completedBikeDetectorCycle observations.CycleSnapshot,
	) {
		_, err := histories.UpdateHistory(
			thingName,
			newCycleStartTime, newCycleEndTime,
			completedPrimarySignalCycle,
			completedSignalProgramCycle,
			completedCycleSecondCycle,
			completedCarDetectorCycle,
			completedBikeDetectorCycle,
		)
		if err != nil {
			return
		}
		predictions.PublishBestPrediction(thingName)
	}

	// Wait forever.
	select {}
}
