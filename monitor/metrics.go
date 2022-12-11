package monitor

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"predictor/calc"
	"predictor/env"
	"predictor/observations"
	"predictor/phases"
	"predictor/predictions"
	"predictor/things"
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
	Deviations    map[string]int `json:"deviations"`    // The time deviation of each prediction from the actual state.
	OldDeviations map[string]int `json:"oldDeviations"` // The time deviation of each prediction from the actual state by the old service.
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
	PredictionAge     *int   `json:"age"`       // The age of the prediction in seconds.
}

// The lock that must be used when writing or reading the metrics file.
// This is to gobally protect concurrent access to the same file.
var metricsFileLock = &sync.Mutex{}

func generateMetrics() Metrics {
	entries := []MetricsEntry{}

	var verifiable int
	var oldVerifiable int
	var correct int
	var oldCorrect int
	var deviations = make(map[string]int)
	var oldDeviations = make(map[string]int)
	var delaySum float64
	var delayCount int

	things.Things.Range(func(key, _ interface{}) bool {
		thingName := key.(string)
		entry := MetricsEntry{ThingName: thingName}
		defer func() { entries = append(entries, entry) }()

		// Get the current state of the thing.
		primarySignalObservation, ok := observations.GetCurrentPrimarySignal(thingName)
		if !ok {
			return true
		}
		entry.ActualColor = &primarySignalObservation.Result

		// Get the last running program of the thing.
		if programObservation, ok := observations.GetCurrentProgram(thingName); ok {
			entry.Program = &programObservation.Result
		}

		// In the monitor, we want to compare the actual color with the predicted color.
		// However, the actual color will always arrive delayed, while the predicted color
		// may be calculated at the current time. Therefore, we need to delay the prediction.
		// Calculate the delay between `phenomenonTime` and `receivedTime` of the observation.
		timeDelay := primarySignalObservation.ReceivedTime.
			Sub(primarySignalObservation.PhenomenonTime)
		delaySum += timeDelay.Seconds()
		delayCount++ // For mean delay calculation.
		// Calculate the current time, subtracting the delay. In this way, we
		// compare a delayed prediction with a delayed observation.
		nowWithDelay := time.Now().Add(-timeDelay)

		if prediction, ok := predictions.GetCurrentPrediction(thingName); ok {
			delayedTimeInPrediction := int(math.Abs(
				nowWithDelay.Sub(prediction.ReferenceTime).Seconds(),
			))
			lenNow := len(prediction.Now)
			lenThen := len(prediction.Then)
			if delayedTimeInPrediction < len(prediction.Now) {
				i := delayedTimeInPrediction % lenNow
				entry.PredictedColor = &prediction.Now[i]
				entry.PredictionQuality = &prediction.NowQuality[i]
			} else {
				i := (delayedTimeInPrediction - lenNow) % lenThen
				entry.PredictedColor = &prediction.Then[i]
				entry.PredictionQuality = &prediction.ThenQuality[i]
			}

			// Look to the left and right to calculate the deviation.
			var minDeviation int = math.MaxInt
			var matchFound bool
			for i, predNow := range prediction.Now {
				tDiff := calc.Abs((delayedTimeInPrediction % lenNow) - i)
				if tDiff < minDeviation && predNow == *entry.ActualColor {
					minDeviation = tDiff
					matchFound = true
				}
			}
			for i, predThen := range prediction.Then {
				tDiff := calc.Abs(((delayedTimeInPrediction - lenNow) % lenThen) - i)
				if tDiff < minDeviation && predThen == *entry.ActualColor {
					minDeviation = tDiff
					matchFound = true
				}
			}
			if matchFound {
				if minDeviation > 30 {
					deviations["30"]++
				} else {
					deviations[fmt.Sprintf("%d", minDeviation)]++
				}
			}
		}
		// Add all missing values to the deviations map.
		for i := 0; i <= 30; i++ {
			if _, ok := deviations[fmt.Sprintf("%d", i)]; !ok {
				deviations[fmt.Sprintf("%d", i)] = 0
			}
		}

		if entry.PredictedColor != nil && entry.ActualColor != nil {
			verifiable++
			if *entry.PredictedColor == *entry.ActualColor {
				correct++
			}
		}

		// Get the old prediction for comparison.
		if oldPrediction, ok := getOldServicePrediction(thingName); ok {
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

			// Look to the left and right to calculate the deviation.
			var minDeviation int = math.MaxInt
			var matchFound bool
			lenValue := len(oldPrediction.Value)
			for i, v := range oldPrediction.Value {
				isGreen := v >= oldPrediction.GreentimeThreshold
				var predNow byte
				if isGreen {
					predNow = phases.Green
				} else {
					predNow = phases.Red
				}
				tDiff := calc.Abs((delayedTimeInPrediction % lenValue) - i)
				if tDiff < minDeviation && predNow == *entry.ActualColor {
					minDeviation = tDiff
					matchFound = true
				}
			}
			if matchFound {
				if minDeviation > 30 {
					oldDeviations["30"]++
				} else {
					oldDeviations[fmt.Sprintf("%d", minDeviation)]++
				}
			}
		}

		// Add all missing values to the deviations map.
		for i := 0; i <= 30; i++ {
			if _, ok := oldDeviations[fmt.Sprintf("%d", i)]; !ok {
				oldDeviations[fmt.Sprintf("%d", i)] = 0
			}
		}

		if entry.OldPrediction != nil && entry.ActualColor != nil {
			oldVerifiable++
			if *entry.OldPrediction == *entry.ActualColor {
				oldCorrect++
			}
		}

		// Get the age of the prediction.
		if lastPredictionTime, ok := predictions.GetLastPredictionTime(thingName); ok {
			age := int(time.Since(lastPredictionTime).Abs().Seconds())
			entry.PredictionAge = &age
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
		OldDeviations: oldDeviations,
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
	file, err := os.Create(fmt.Sprintf("%s/metrics.json", env.StaticPath))
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
