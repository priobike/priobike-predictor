package monitor

import (
	"encoding/json"
	"io/ioutil"
	"os"
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

// Write a status file for each signal group.
func WriteStatusForEachSG() {
	things.Things.Range(func(key, value interface{}) bool {
		thingName := key.(string)
		thing := value.(things.Thing)

		// Create the status summary.
		status := SGStatus{
			StatusUpdateTime: time.Now().Unix(),
			ThingName:        thing.Name,
		}

		// Get the prediction for the signal group.
		prediction, ok := predictions.GetCurrentPrediction(thingName)
		if ok {
			avg := prediction.AverageQuality() / 100
			status.PredictionQuality = &avg
			t := prediction.ReferenceTime.Unix()
			status.PredictionTime = &t
		}

		// Write the status update to a json file.
		statusJson, err := json.Marshal(status)
		if err != nil {
			log.Error.Println("Error marshalling status:", err)
			return true
		}
		path := env.StaticPath + "/status/" + thing.Topic()
		if err := ioutil.WriteFile(path+"/status.json", statusJson, 0644); err != nil {
			// If the path contains a directory that does not exist, create it.
			// But don't create a folder for the file itself.
			if err := os.MkdirAll(path, 0755); err != nil {
				log.Error.Println("Error creating directory for status file:", err)
				return true
			}
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
