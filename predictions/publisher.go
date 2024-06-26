package predictions

import (
	"predictor/log"
	"predictor/things"
	"sync"
	"sync/atomic"
	"time"
)

// The most recent prediction by their thing name.
var Current = &sync.Map{}

// Get the most recent prediction for a given thing.
func GetCurrentPrediction(thingName string) (Prediction, bool) {
	prediction, ok := Current.Load(thingName)
	if !ok {
		return Prediction{}, false
	}
	return prediction.(Prediction), true
}

// Get the number of predictions that are currently stored.
func CountPredictions() int {
	var count int
	Current.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Locks that must be used when making a prediction.
var predictionLocks = &sync.Map{}

// The times at which a prediction was published for a given thing.
var Times = &sync.Map{}

// Get the last time a prediction was published for a given thing.
func GetLastPredictionTime(thingName string) (time.Time, bool) {
	t, ok := Times.Load(thingName)
	if !ok {
		return time.Time{}, false
	}
	return t.(time.Time), true
}

var PredictionsChecked uint64 = 0
var PredictionsDiscarded uint64 = 0
var PredictionsPublished uint64 = 0

// Build a prediction for a given history.
func PublishBestPrediction(thingName string) {
	// Acquire the corresponding lock to avoid race conditions for the same thing.
	lock, _ := predictionLocks.LoadOrStore(thingName, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()

	atomic.AddUint64(&PredictionsChecked, 1)

	prediction, err := predict(thingName)
	if err != nil {
		atomic.AddUint64(&PredictionsDiscarded, 1)
		return
	}

	// Check if this prediction equals the last prediction.
	// This avoids writing the same prediction (costly operation)
	// multiple times to the MQTT broker.
	if lastPrediction, ok := GetCurrentPrediction(thingName); ok {
		if prediction.Equals(lastPrediction) {
			atomic.AddUint64(&PredictionsDiscarded, 1)
			return
		}

		// Avoid publishing new predictions too rapidly.
		if prediction.WithinTimeOfSeconds(1*time.Second, lastPrediction) {
			atomic.AddUint64(&PredictionsDiscarded, 1)
			return
		}
	}

	err = publish(prediction)
	if err != nil {
		log.Error.Printf("Could not publish prediction to MQTT: %s", err)
		atomic.AddUint64(&PredictionsDiscarded, 1)
		return
	}

	Current.Store(prediction.ThingName, prediction)
	Times.Store(thingName, time.Now())

	atomic.AddUint64(&PredictionsPublished, 1)
	if (PredictionsPublished%1000) == 0 && PredictionsPublished > 0 {
		log.Info.Printf(
			"Predictions requested %d, published %d, canceled %d",
			PredictionsChecked, PredictionsPublished, PredictionsDiscarded,
		)
	}
}

// Publish best predictions for all things.
func PublishAllBestPredictions() {
	var wg sync.WaitGroup
	things.Things.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			PublishBestPrediction(key.(string))
		}()
		return true
	})
	wg.Wait()
}

// Publish best predictions for all things periodically.
func PublishAllBestPredictionsPeriodically() {
	for {
		PublishAllBestPredictions()
		time.Sleep(500 * time.Millisecond)
	}
}
