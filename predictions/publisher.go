package predictions

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/env"
	"predictor/log"
	"predictor/things"
	"sync"
	"sync/atomic"
	"time"
)

// The most recent prediction by their thing name.
var current = &sync.Map{}

// Get the most recent prediction for a given thing.
func GetCurrentPrediction(thingName string) (Prediction, bool) {
	prediction, ok := current.Load(thingName)
	if !ok {
		return Prediction{}, false
	}
	return prediction.(Prediction), true
}

// Locks that must be used when making a prediction.
var predictionLocks = &sync.Map{}

// The times at which a prediction was published for a given thing.
var times = &sync.Map{}

// Get the last time a prediction was published for a given thing.
func GetLastPredictionTime(thingName string) (time.Time, bool) {
	t, ok := times.Load(thingName)
	if !ok {
		return time.Time{}, false
	}
	return t.(time.Time), true
}

var requested uint64 = 0
var canceled uint64 = 0
var processed uint64 = 0

// Build a prediction for a given history.
func PublishBestPrediction(thingName string) {
	// Acquire the corresponding lock to avoid race conditions for the same thing.
	lock, _ := predictionLocks.LoadOrStore(thingName, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()

	atomic.AddUint64(&requested, 1)

	prediction, err := predict(thingName)
	if err != nil {
		atomic.AddUint64(&canceled, 1)
		return
	}

	// Check if this prediction equals the last prediction.
	// This avoids writing the same prediction (costly operation)
	// multiple times to the file system and the MQTT broker.
	if lastPrediction, ok := GetCurrentPrediction(thingName); ok {
		if prediction.Equals(lastPrediction) {
			atomic.AddUint64(&canceled, 1)
			return
		}
	}

	// Write the prediction to a file.
	path := fmt.Sprintf("%s/predictions/%s.json", env.StaticPath, thingName)
	if prediction.ProgramId != nil {
		path = fmt.Sprintf("%s/predictions/%s-P%d.json", env.StaticPath, thingName, *prediction.ProgramId)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Error.Printf("Could not open file %s: %s", path, err)
		atomic.AddUint64(&canceled, 1)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(prediction)
	if err != nil {
		log.Error.Printf("Could not write to file %s: %s", path, err)
		atomic.AddUint64(&canceled, 1)
		return
	}

	err = prediction.publish()
	if err != nil {
		log.Error.Printf("Could not publish prediction to MQTT: %s", err)
		atomic.AddUint64(&canceled, 1)
		return
	}

	current.Store(prediction.ThingName, prediction)
	times.Store(thingName, time.Now())

	atomic.AddUint64(&processed, 1)
	if (processed%1000) == 0 && processed > 0 {
		log.Info.Printf("Predictions requested %d, processed %d, canceled %d", requested, processed, canceled)
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
