package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/env"
	"predictor/predictions"
	"predictor/things"
	"testing"
	"time"
)

func TestWriteStatusForEachSG(t *testing.T) {
	exampleThing := things.Thing{
		Name: "1337_1", // All other fields are not needed.
	}
	getThingsForSGStatus = func(f func(key, value interface{}) bool) {
		f("1337_1", exampleThing)
	}
	getCurrentPredictionForSGStatus = func(_ string) (predictions.Prediction, bool) {
		return predictions.Prediction{
			ReferenceTime: time.Unix(0, 0),
			NowQuality:    []byte{100, 100, 100},
			ThenQuality:   []byte{100, 100, 100},
		}, true
	}

	tempDir := t.TempDir()
	env.StaticPath = tempDir

	timeBeforeWrite := time.Now().Unix()
	WriteStatusForEachSG()

	expectedFileDir := fmt.Sprintf("%s/status/%s/status.json", tempDir, exampleThing.Topic())
	file, err := os.Open(expectedFileDir)
	if err != nil {
		t.Errorf("status.json could not be read: %s", err.Error())
		t.FailNow()
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var statusFromFile SGStatus
	err = decoder.Decode(&statusFromFile)
	if err != nil {
		t.Errorf("error decoding sg status from file: %s", err.Error())
		t.FailNow()
	}

	if timeBeforeWrite > statusFromFile.StatusUpdateTime {
		t.Errorf("time in status update was not updated correctly")
		t.FailNow()
	}
	if statusFromFile.PredictionQuality == nil || *statusFromFile.PredictionQuality != 1 {
		t.Errorf("failure to calculate correct prediction quality")
		t.FailNow()
	}
	if statusFromFile.PredictionTime == nil || *statusFromFile.PredictionTime != 0 {
		t.Errorf("wrong prediction time")
		t.FailNow()
	}
	if statusFromFile.ThingName != "1337_1" {
		t.Errorf("wrong thing name")
	}
}
