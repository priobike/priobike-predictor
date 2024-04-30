package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/env"
	"predictor/predictions"
	"testing"
	"time"
)

func TestWriteSummary(t *testing.T) {
	getNumberOfThings = func() int { return 1 }
	getNumberOfPredictions = func() int { return 1 }
	getCurrentPredictions = func(f func(key, value interface{}) bool) {
		f("mock-topic", predictions.Prediction{
			ReferenceTime: time.Unix(0, 0),
			NowQuality:    []byte{100, 100, 100, 100},
			ThenQuality:   []byte{100, 100, 100, 100},
		})
	}

	tempDir := t.TempDir()
	env.StaticPath = tempDir
	WriteSummary()

	expectedFilePath := fmt.Sprintf("%s/status/status.json", tempDir)
	file, err := os.Open(expectedFilePath)
	if err != nil {
		t.Errorf("failed to open status.json file: %s", err.Error())
		t.FailNow()
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var summary StatusSummary
	err = decoder.Decode(&summary)
	if err != nil {
		t.Errorf("failed to decode summary: %s", err.Error())
		t.FailNow()
	}

	if summary.AveragePredictionQuality == nil || *summary.AveragePredictionQuality != 1 {
		t.Errorf("wrong avg prediction quality")
		t.FailNow()
	}
	if summary.NumThings != 1 || summary.NumPredictions != 1 || summary.NumBadPredictions != 0 {
		t.Errorf("expected 1 thing, 1 prediction, and 0 bad predictions")
		t.FailNow()
	}
}
