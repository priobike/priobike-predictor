package predictions

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/log"
	"predictor/things"
	"sync"
	"sync/atomic"
	"time"
)

// The most recent prediction by their thing name.
var current = &sync.Map{}

// The time when the last prediction was requested, by their thing name.
var lastRequested = &sync.Map{}

// Get the most recent prediction for a given thing.
func Current(thingName string) (Prediction, error) {
	prediction, ok := current.Load(thingName)
	if !ok {
		return Prediction{}, fmt.Errorf("no prediction for thing %s", thingName)
	}
	return prediction.(Prediction), nil
}

// Locks that must be used when making a prediction.
var predictionLocks = &sync.Map{}

var requested uint64 = 0
var canceled uint64 = 0
var processed uint64 = 0

// Build a prediction for a given history.
func PublishBestPrediction(thingName string) {
	// Acquire the corresponding lock to avoid race conditions for the same thing.
	lock, _ := predictionLocks.LoadOrStore(thingName, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()

	lastRequested.Store(thingName, time.Now())

	atomic.AddUint64(&requested, 1)
	if (requested%1000) == 0 && requested > 0 {
		log.Info.Printf("Predictions requested %d, processed %d, canceled %d", requested, processed, canceled)
	}

	prediction, err := predict(thingName)
	if err != nil {
		atomic.AddUint64(&canceled, 1)
		return
	}

	// Check if this prediction equals the last prediction.
	// This avoids writing the same prediction (costly operation)
	// multiple times to the file system and the MQTT broker.
	lastPrediction, err := Current(thingName)
	if err == nil && prediction.Equals(lastPrediction) {
		atomic.AddUint64(&canceled, 1)
		return
	}

	// Write the prediction to a file.
	path := fmt.Sprintf("%s/predictions/%s.json", staticPath, thingName)
	if prediction.ProgramId != nil {
		path = fmt.Sprintf("%s/predictions/%s-P%d.json", staticPath, thingName, *prediction.ProgramId)
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
	atomic.AddUint64(&processed, 1)
}

// Publish best predictions for all things.
func PublishAllBestPredictions() {
	log.Info.Println("Publishing predictions for all things...")
	var wg sync.WaitGroup
	things.Things.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Check that the prediction wasn't recently requested.
			// This avoids making too many predictions for the same thing.
			last, ok := lastRequested.Load(key.(string))
			if ok && time.Since(last.(time.Time)) < 60*time.Second {
				return
			}
			PublishBestPrediction(key.(string))
		}()
		return true
	})
	wg.Wait()
	log.Info.Println("Done publishing predictions for all things.")
}
