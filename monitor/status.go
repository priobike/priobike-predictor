package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"predictor/env"
	"predictor/log"
	"predictor/predictions"
	"predictor/things"
	"time"
)

// A status summary of all predictions that is written to json.
type StatusSummary struct {
	// The time of the status update.
	StatusUpdateTime int64 `json:"status_update_time"`
	// The number of things.
	NumThings int `json:"num_things"`
	// The number of predictions.
	NumPredictions int `json:"num_predictions"`
	// The number of predictions with quality <= 0.5.
	NumBadPredictions int `json:"num_bad_predictions"`
	// The time of the most recent prediction.
	MostRecentPredictionTime *int64 `json:"most_recent_prediction_time"`
	// The time of the oldest prediction.
	OldestPredictionTime *int64 `json:"oldest_prediction_time"`
	// The average prediction quality.
	AveragePredictionQuality *float64 `json:"average_prediction_quality"`
}

// Interfaces to other packages.
var (
	getNumberOfThings      = things.CountThings
	getNumberOfPredictions = predictions.CountPredictions
	getCurrentPredictions  = predictions.Current.Range
)

// Create a summary of the predictions, i.e. whether they are up to date.
// Write the result to a static directory as json.
func WriteSummary() {
	numThings := getNumberOfThings()
	numPredictions := getNumberOfPredictions()

	var mostRecentPredictionTime *int64 = nil
	var oldestPredictionTime *int64 = nil
	getCurrentPredictions(func(key, value interface{}) bool {
		prediction := value.(predictions.Prediction)
		t := prediction.ReferenceTime.Unix()
		if mostRecentPredictionTime == nil || t > *mostRecentPredictionTime {
			mostRecentPredictionTime = &t
		}
		if oldestPredictionTime == nil || t < *oldestPredictionTime {
			oldestPredictionTime = &t
		}
		return true
	})

	// Calculate the average prediction quality and the number of bad predictions.
	var averagePredictionQuality *float64 = nil
	numBadPredictions := 0
	if numPredictions > 0 {
		var sum float64 = 0
		getCurrentPredictions(func(key, value interface{}) bool {
			prediction := value.(predictions.Prediction)
			quality := prediction.AverageQuality() / 100
			if quality <= 0.5 {
				numBadPredictions++
			}
			if (quality < 0) || (quality > 1) {
				return true
			}
			sum += quality
			return true
		})
		average := sum / float64(numPredictions)
		averagePredictionQuality = &average
	}

	// Write the status update to a json file.
	summary := StatusSummary{
		StatusUpdateTime:         time.Now().Unix(),
		NumThings:                numThings,
		NumPredictions:           numPredictions,
		NumBadPredictions:        numBadPredictions,
		MostRecentPredictionTime: mostRecentPredictionTime,
		OldestPredictionTime:     oldestPredictionTime,
		AveragePredictionQuality: averagePredictionQuality,
	}

	// Write the status update to the file.
	filePath := fmt.Sprintf("%s/status/status.json", env.StaticPath)
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		log.Error.Println("Error creating directory for status summary file:", err)
		return
	}
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Error.Println("Error writing summary status.json: ", err)
		return
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(summary)
	if err != nil {
		log.Error.Println("Error marshaling to summary status.json: ", err)
		return
	}
}

func UpdateStatusSummaryPeriodically() {
	for {
		time.Sleep(30 * time.Second)
		WriteSummary()
	}
}
