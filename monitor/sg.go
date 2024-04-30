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
type SGStatus struct {
	// The time of the status update.
	StatusUpdateTime int64 `json:"status_update_time"`
	// The name of the thing.
	ThingName string `json:"thing_name"`
	// The current prediction quality, if there is a prediction.
	PredictionQuality *float64 `json:"prediction_quality"`
	// The unix time of the last prediction, if there is a prediction.
	PredictionTime *int64 `json:"prediction_time"`
}

// Interface to overwrite for testing purposes.
var getThingsForSGStatus = things.Things.Range
var getCurrentPredictionForSGStatus = predictions.GetCurrentPrediction

// Write a status file for each signal group.
func WriteStatusForEachSG() {
	getThingsForSGStatus(func(key, value interface{}) bool {
		thingName := key.(string)
		thing := value.(things.Thing)

		// Create the status summary.
		status := SGStatus{
			StatusUpdateTime: time.Now().Unix(),
			ThingName:        thing.Name,
		}

		// Get the prediction for the signal group.
		prediction, ok := getCurrentPredictionForSGStatus(thingName)
		if ok {
			avg := prediction.AverageQuality() / 100
			status.PredictionQuality = &avg
			t := prediction.ReferenceTime.Unix()
			status.PredictionTime = &t
		}

		// Write the status update to a json file.
		filePath := fmt.Sprintf("%s/status/%s/status.json", env.StaticPath, thing.Topic())
		// Make sure the directory exists, otherwise create it.
		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			log.Error.Println("Error creating directory for status file:", err)
			return true
		}
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Error.Println("Error writing status.json: ", err)
			return true
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		err = encoder.Encode(status)
		if err != nil {
			log.Error.Println("Error marshaling to status.json: ", err)
			return true
		}
		return true
	})
}

func UpdateSGStatusPeriodically() {
	for {
		time.Sleep(30 * time.Second)
		WriteStatusForEachSG()
	}
}
