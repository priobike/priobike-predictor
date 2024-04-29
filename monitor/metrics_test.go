package monitor

import (
	"fmt"
	"os"
	"predictor/env"
	"predictor/observations"
	"predictor/predictions"
	"predictor/things"
	"strconv"
	"strings"
	"testing"
	"time"
)

func prepareMocks() {
	getAllThingsForMetrics = func(f func(key, value interface{}) bool) {
		f("1337_1", things.Thing{}) // Thing can be empty, only name is needed
	}
	getCurrentPrimarySignalForMetrics = func(thingName string) (observations.Observation, bool) {
		return observations.Observation{
			Result:         4, // RedAmber
			ReceivedTime:   time.Unix(1, 0),
			PhenomenonTime: time.Unix(0, 0), // 1 second delay
		}, true
	}
	getCurrentProgramForMetrics = func(thingName string) (observations.Observation, bool) {
		return observations.Observation{
			Result:         10, // Program 10
			ReceivedTime:   time.Unix(1, 0),
			PhenomenonTime: time.Unix(0, 0), // 1 second delay
		}, true
	}
	getCurrentPredictionForMetrics = func(thingName string) (predictions.Prediction, bool) {
		return predictions.Prediction{
			ThingName:   "1337_1",
			Now:         []byte{1, 1, 1, 1, 1, 3, 3, 3, 3, 3},
			NowQuality:  []byte{100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
			Then:        []byte{4, 4, 4, 4, 4, 4, 4, 4, 4, 4},
			ThenQuality: []byte{100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
			// Should point to the then array (now is >50yrs after)
			// -> should result in a 0 seconds deviation from the actual data
			ReferenceTime: time.Unix(0, 0),
		}, true
	}
	getLastPredictionTimeForMetrics = func(thingName string) (time.Time, bool) {
		return time.Unix(0, 0), true
	}
	getObservationsReceivedByTopic = func(callback func(dsType string, count uint64)) {
		callback("primary_signal", 1)
		callback("signal_program", 1)
		callback("cycle_second", 1)
		callback("detector_bike", 0)
		callback("detector_car", 0)
	}
	getObservationsReceived = func() uint64 {
		return 1
	}
	getObservationsProcessed = func() uint64 {
		return 1
	}
	getObservationsDiscarded = func() uint64 {
		return 1
	}
	getHistoryUpdatesRequested = func() uint64 {
		return 1
	}
	getHistoryUpdatesProcessed = func() uint64 {
		return 1
	}
	getHistoryUpdatesDiscarded = func() uint64 {
		return 1
	}
	getPredictionsChecked = func() uint64 {
		return 1
	}
	getPredictionsPublished = func() uint64 {
		return 1
	}
	getPredictionsDiscarded = func() uint64 {
		return 1
	}
}

func TestGenerateMetrics(t *testing.T) {
	prepareMocks()
	metrics := generateMetrics()

	if len(metrics.Entries) != 1 {
		t.Errorf("metrics has more or fewer entries than expected")
		t.FailNow()
	}
	entry := metrics.Entries[0]
	// Check the entry
	if entry.ActualColor == nil || entry.PredictedColor == nil {
		t.Errorf("actual color or predicted color is nil")
		t.FailNow()
	}
	if *entry.ActualColor != 4 || *entry.PredictedColor != 4 {
		t.Errorf("unexpected prediction color and actual color")
		t.FailNow()
	}

	// Check the metrics
	if metrics.Deviations["0"] != 1 {
		t.Errorf("deviations are not correct: %v", metrics.Deviations)
		t.FailNow()
	}
	if metrics.MeanMsgDelay != 1 {
		t.Errorf("unexpected message delay: %f", metrics.MeanMsgDelay)
		t.FailNow()
	}
	if metrics.Correct != 1 || metrics.Verifiable != 1 {
		t.Errorf("prediction for the given example should be both verifiable and correct")
		t.FailNow()
	}
}

func TestGeneratePrometheusMetrics(t *testing.T) {
	prepareMocks()
	baseMetrics := generateMetrics()
	prometheusMetrics := generatePrometheusMetrics(baseMetrics)

	var search = func(metricName string, expectedValue interface{}) bool {
		for _, metric := range prometheusMetrics {
			if !strings.Contains(metric, metricName) {
				continue
			}
			// Found the metric we are searching for
			strParts := strings.Split(metric, " ")
			if len(strParts) < 2 {
				t.Errorf("something is wrong with the metric's formatting: %s", metric)
				t.FailNow()
			}
			valueAsStr := strParts[len(strParts)-1]
			var actualValue interface{}
			// Try to parse as int, then as float
			actualValue, err := strconv.Atoi(valueAsStr)
			if err != nil {
				actualValue, err = strconv.ParseFloat(valueAsStr, 64)
				if err != nil {
					t.Errorf("could not parse value: %s", valueAsStr)
					t.FailNow()
				}
			}
			// Since we only compare floats or ints, its ok to use ==
			return expectedValue == actualValue
		}
		return false
	}

	if !search("predictor_verifiable", 1) || !search("predictor_correct", 1) {
		t.Errorf("prediction for the given example should be both verifiable and correct")
		t.FailNow()
	}
	if !search("predictor_mean_msg_delay", 1.0) {
		t.Errorf("unexpected message delay")
		t.FailNow()
	}
	if !search("predictor_observations{action=\"received\"}", 1) || //
		!search("predictor_observations{action=\"processed\"}", 1) || //
		!search("predictor_observations{action=\"discarded\"}", 1) {
		t.Errorf("unexpected metrics value")
		t.FailNow()
	}
	if !search("predictor_histories{action=\"requested\"}", 1) || //
		!search("predictor_histories{action=\"processed\"}", 1) || //
		!search("predictor_histories{action=\"discarded\"}", 1) || //
		!search("predictor_predictions{action=\"checked\"}", 1) || //
		!search("predictor_predictions{action=\"published\"}", 1) || //
		!search("predictor_predictions{action=\"discarded\"}", 1) {
		t.Errorf("unexpected metrics value")
		t.FailNow()
	}
	if !search("predictor_deviation{bucket=\"00\"}", 1) {
		t.Errorf("unexpected metrics value")
		t.FailNow()
	}
}

func TestMetricsFile(t *testing.T) {
	prepareMocks()

	tempDir := t.TempDir()
	env.StaticPath = tempDir

	UpdateMetricsFiles()

	filePath := fmt.Sprintf("%s/metrics.json", tempDir)
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("metrics file could not be opened")
		t.FailNow()
	}
	fileAsStr := string(fileContent)
	metrics := strings.Split(fileAsStr, "\n")

	// Just a simple check if there are metrics, the logic
	// is checked in the other tests
	if len(metrics) == 0 {
		t.Errorf("no metrics found")
	}
}
