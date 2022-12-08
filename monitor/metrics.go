package monitor

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"predictor/observations"
	"predictor/phases"
	"predictor/predictions"
	"predictor/things"
	"predictor/util"
	"sort"
	"sync"
	"time"
)

type Metrics struct {
	Entries       []MetricsEntry `json:"entries"`       // Metrics for each traffic light.
	Verifiable    int            `json:"verifiable"`    // The number of traffic lights sending data and having a prediction.
	OldVerifiable int            `json:"oldVerifiable"` // The number of traffic lights sending data and having a prediction by the old service.
	Correct       int            `json:"correct"`       // The number of traffic lights that were predicted correctly.
	OldCorrect    int            `json:"oldCorrect"`    // The number of traffic lights that were predicted correctly by the old service.
	Deviations    map[int]int    `json:"deviations"`    // The time deviation of each prediction from the actual state.
	MeanDeviation float64        `json:"meanDev"`       // The mean of the deviations.
	MeanMsgDelay  float64        `json:"meanMsgDelay"`  // The mean message delay.
}

type MetricsEntry struct {
	ThingName         string `json:"name"`      // The name of the thing.
	ActualColor       *byte  `json:"actual"`    // The actual color of the traffic light.
	PredictedColor    *byte  `json:"predicted"` // The predicted color of the traffic light.
	PredictionQuality *byte  `json:"quality"`   // The quality of the prediction.
	OldPrediction     *byte  `json:"old"`       // The state of the prediction from the old prediction service.
	Program           *byte  `json:"program"`   // The program that the traffic light is currently running.
}

// The lock that must be used when writing or reading the metrics file.
// This is to gobally protect concurrent access to the same file.
var metricsFileLock = &sync.Mutex{}

// The maximum age of a `primary_signal` observation before it is no longer considered valid.
const maxPrimarySignalAge = 300 * time.Second

func generateMetrics() Metrics {
	entries := []MetricsEntry{}

	var verifiable int
	var oldVerifiable int
	var correct int
	var oldCorrect int
	var delaySum float64
	var delayCount int
	var deviations = make(map[int]int)

	things.Things.Range(func(key, _ interface{}) bool {
		thingName := key.(string)
		entry := MetricsEntry{ThingName: thingName}
		defer func() { entries = append(entries, entry) }()

		// Get the current state of the thing.
		primarySignalCycle, err := observations.GetPrimarySignalCycle(thingName)
		if err != nil {
			return true
		}
		currentState, err := primarySignalCycle.GetMostRecentObservation()
		if err != nil || time.Since(currentState.PhenomenonTime) > maxPrimarySignalAge {
			return true
		}
		entry.ActualColor = &currentState.Result

		// Get the last running program of the thing.
		program, hasProgram := observations.GetCurrentProgram(thingName)
		if hasProgram {
			entry.Program = &program
		}

		// Get the predicted state of the thing.
		prediction, err := predictions.Current(thingName)
		if err != nil {
			return true
		}

		// Calculate the predicted color.
		lenNow := len(prediction.Now)
		lenThen := len(prediction.Then)
		// In the monitor, we want to compare the actual color with the predicted color.
		// However, the actual color will always arrive delayed, while the predicted color
		// may be calculated at the current time. Therefore, we need to delay the prediction.
		// Calculate the delay between `phenomenonTime` and `receivedTime` of the observation.
		timeDelay := currentState.ReceivedTime.Sub(currentState.PhenomenonTime)
		delaySum += timeDelay.Seconds()
		delayCount++ // For mean delay calculation.
		// Calculate the current time, subtracting the delay. In this way, we
		// compare a delayed prediction with a delayed observation.
		nowWithDelay := time.Now().Add(-timeDelay)
		delayedTimeInPrediction := int(math.Abs(nowWithDelay.Sub(prediction.ReferenceTime).Seconds()))
		if delayedTimeInPrediction < len(prediction.Now) {
			i := delayedTimeInPrediction % lenNow
			entry.PredictedColor = &prediction.Now[i]
			entry.PredictionQuality = &prediction.NowQuality[i]
		} else {
			i := (delayedTimeInPrediction - lenNow) % lenThen
			entry.PredictedColor = &prediction.Then[i]
			entry.PredictionQuality = &prediction.ThenQuality[i]
		}

		if entry.PredictedColor != nil && entry.ActualColor != nil {
			verifiable++
			if *entry.PredictedColor == *entry.ActualColor {
				correct++
			}
		}

		// Look to the left and right to calculate the deviation.
		var minDeviation int = math.MaxInt
		var matchFound bool
		for i, predNow := range prediction.Now {
			tDiff := util.Abs((delayedTimeInPrediction % lenNow) - i)
			if tDiff < minDeviation && predNow == *entry.ActualColor {
				minDeviation = tDiff
				matchFound = true
			}
		}
		for i, predThen := range prediction.Then {
			tDiff := util.Abs(((delayedTimeInPrediction - lenNow) % lenThen) - i)
			if tDiff < minDeviation && predThen == *entry.ActualColor {
				minDeviation = tDiff
				matchFound = true
			}
		}
		if matchFound {
			deviations[minDeviation]++
		}

		// Get the old prediction for comparison.
		oldPrediction, err := getOldServicePrediction(thingName)
		if err == nil {
			delayedTimeInPrediction := int(math.Abs(nowWithDelay.Sub(oldPrediction.StartTime).Seconds()))
			if delayedTimeInPrediction > 0 && delayedTimeInPrediction < len(oldPrediction.Value) {
				value := oldPrediction.Value[delayedTimeInPrediction]
				isGreen := value >= oldPrediction.GreentimeThreshold
				if isGreen {
					green := phases.Green
					entry.OldPrediction = &green
				} else {
					red := phases.Red
					entry.OldPrediction = &red
				}
			}
		}

		if entry.OldPrediction != nil && entry.ActualColor != nil {
			oldVerifiable++
			if *entry.OldPrediction == *entry.ActualColor {
				oldCorrect++
			}
		}

		return true
	})

	// Sort the entries by thing name.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ThingName < entries[j].ThingName
	})

	var meanDev float64
	if verifiable > 0 {
		var sumDev float64 = 0
		for _, v := range deviations {
			sumDev += float64(v)
		}
		meanDev = sumDev / float64(verifiable)
	}
	var meanMsgDelay float64
	if delayCount > 0 {
		meanMsgDelay = delaySum / float64(delayCount)
	}

	return Metrics{
		Entries:       entries,
		Deviations:    deviations,
		Verifiable:    verifiable,
		OldVerifiable: oldVerifiable,
		Correct:       correct,
		OldCorrect:    oldCorrect,
		MeanDeviation: meanDev,
		MeanMsgDelay:  meanMsgDelay,
	}
}

func UpdateMetricsFile() {
	metrics := generateMetrics()

	// Write the metrics to a file.
	metricsFileLock.Lock()
	defer metricsFileLock.Unlock()

	// Write the json files into a json file (without ioutil).
	jsonBytes, err := json.Marshal(metrics)
	if err != nil {
		panic(err)
	}
	file, err := os.Create(fmt.Sprintf("%s/metrics.json", staticPath))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = file.Write(jsonBytes)
	if err != nil {
		panic(err)
	}
}

// Build the metrics file periodically.
func UpdateMetricsFilePeriodically() {
	for {
		time.Sleep(1 * time.Second)
		UpdateMetricsFile()
	}
}
