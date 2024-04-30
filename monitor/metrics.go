package monitor

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"predictor/calc"
	"predictor/env"
	"predictor/histories"
	"predictor/observations"
	"predictor/predictions"
	"predictor/things"
	"sort"
	"strings"
	"sync"
	"time"
)

type Metrics struct {
	Entries       []MetricsEntry `json:"entries"`       // Metrics for each traffic light.
	Verifiable    int            `json:"verifiable"`    // The number of traffic lights sending data and having a prediction.
	OldVerifiable int            `json:"oldVerifiable"` // The number of traffic lights sending data and having a prediction by the old service.
	Correct       int            `json:"correct"`       // The number of traffic lights that were predicted correctly.
	Deviations    map[string]int `json:"deviations"`    // The time deviation of each prediction from the actual state.
	MeanDeviation float64        `json:"meanDev"`       // The mean of the deviations.
	MeanMsgDelay  float64        `json:"meanMsgDelay"`  // The mean message delay.
}

type MetricsEntry struct {
	ThingName         string `json:"name"`      // The name of the thing.
	ActualColor       *byte  `json:"actual"`    // The actual color of the traffic light.
	PredictedColor    *byte  `json:"predicted"` // The predicted color of the traffic light.
	PredictionQuality *byte  `json:"quality"`   // The quality of the prediction.
	Program           *byte  `json:"program"`   // The program that the traffic light is currently running.
	PredictionAge     *int   `json:"age"`       // The age of the prediction in seconds.
}

// The lock that must be used when writing or reading the metrics file.
// This is to gobally protect concurrent access to the same file.
var metricsFileLock = &sync.Mutex{}

// Interfaces to other packages.
var getAllThingsForMetrics = things.Things.Range                                    // pointer ref
var getCurrentPrimarySignalForMetrics = observations.GetCurrentPrimarySignal        // func ref
var getCurrentProgramForMetrics = observations.GetCurrentProgram                    // func ref
var getCurrentPredictionForMetrics = predictions.GetCurrentPrediction               // func ref
var getLastPredictionTimeForMetrics = predictions.GetLastPredictionTime             // func ref
var getObservationsReceivedByTopic = observations.ObservationsReceivedByTopic.Range // pointer ref
var getObservationsReceived = func() uint64 { return observations.ObservationsReceived }
var getObservationsProcessed = func() uint64 { return observations.ObservationsProcessed }
var getObservationsDiscarded = func() uint64 { return observations.ObservationsDiscarded }
var getHistoryUpdatesRequested = func() uint64 { return histories.HistoryUpdatesRequested }
var getHistoryUpdatesProcessed = func() uint64 { return histories.HistoryUpdatesProcessed }
var getHistoryUpdatesDiscarded = func() uint64 { return histories.HistoryUpdatesDiscarded }
var getPredictionsChecked = func() uint64 { return predictions.PredictionsChecked }
var getPredictionsPublished = func() uint64 { return predictions.PredictionsPublished }
var getPredictionsDiscarded = func() uint64 { return predictions.PredictionsDiscarded }

func generateMetrics() Metrics {
	entries := []MetricsEntry{}

	var verifiable int
	var correct int
	var deviations = make(map[string]int)
	var delaySum float64
	var delayCount int

	getAllThingsForMetrics(func(key, _ interface{}) bool {
		thingName := key.(string)
		entry := MetricsEntry{ThingName: thingName}
		defer func() { entries = append(entries, entry) }()

		// Get the current state of the thing.
		primarySignalObservation, ok := getCurrentPrimarySignalForMetrics(thingName)
		if !ok {
			return true
		}
		entry.ActualColor = &primarySignalObservation.Result

		// Get the last running program of the thing.
		if programObservation, ok := getCurrentProgramForMetrics(thingName); ok {
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

		if prediction, ok := getCurrentPredictionForMetrics(thingName); ok {
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
				if minDeviation > 10 {
					deviations["10"]++
				} else {
					deviations[fmt.Sprintf("%d", minDeviation)]++
				}
			}
		}
		// Add all missing values to the deviations map.
		for i := 0; i <= 10; i++ {
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

		// Get the age of the prediction.
		if lastPredictionTime, ok := getLastPredictionTimeForMetrics(thingName); ok {
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
		Verifiable:    verifiable,
		Correct:       correct,
		MeanDeviation: meanDev,
		MeanMsgDelay:  meanMsgDelay,
	}
}

// Convert the metrics to prometheus metrics.
func generatePrometheusMetrics(m Metrics) []string {
	var lines []string

	// Add the metrics.
	lines = append(lines, fmt.Sprintf("predictor_verifiable %d", m.Verifiable))
	lines = append(lines, fmt.Sprintf("predictor_correct %d", m.Correct))
	lines = append(lines, fmt.Sprintf("predictor_mean_deviation %f", m.MeanDeviation))
	lines = append(lines, fmt.Sprintf("predictor_mean_msg_delay %f", m.MeanMsgDelay))

	// Add metrics for the observations.
	lines = append(lines, fmt.Sprintf("predictor_observations{action=\"received\"} %d", getObservationsReceived()))
	lines = append(lines, fmt.Sprintf("predictor_observations{action=\"processed\"} %d", getObservationsProcessed()))
	lines = append(lines, fmt.Sprintf("predictor_observations{action=\"discarded\"} %d", getObservationsDiscarded()))
	getObservationsReceivedByTopic(func(k, v interface{}) bool {
		dsType := k.(string)
		count := v.(uint64)
		lines = append(lines, fmt.Sprintf("predictor_observations_by_topic{topic=\"%s\"} %d", dsType, count))
		return true
	})

	// Add metrics for the histories.
	lines = append(lines, fmt.Sprintf("predictor_histories{action=\"requested\"} %d", getHistoryUpdatesRequested()))
	lines = append(lines, fmt.Sprintf("predictor_histories{action=\"processed\"} %d", getHistoryUpdatesProcessed()))
	lines = append(lines, fmt.Sprintf("predictor_histories{action=\"discarded\"} %d", getHistoryUpdatesDiscarded()))

	// Add metrics for the predictions.
	lines = append(lines, fmt.Sprintf("predictor_predictions{action=\"checked\"} %d", getPredictionsChecked()))
	lines = append(lines, fmt.Sprintf("predictor_predictions{action=\"published\"} %d", getPredictionsPublished()))
	lines = append(lines, fmt.Sprintf("predictor_predictions{action=\"discarded\"} %d", getPredictionsDiscarded()))

	for bucket, value := range m.Deviations {
		// Add with trailing 0s to make the graph look nicer.
		if len(bucket) == 1 {
			bucket = fmt.Sprintf("0%s", bucket)
		}
		lines = append(lines, fmt.Sprintf("predictor_deviation{bucket=\"%s\"} %d", bucket, value))
	}

	// Sort alphabetically.
	sort.Strings(lines)

	return lines
}

func UpdateMetricsFiles() {
	jsonMetrics := generateMetrics()
	prometheusMetrics := generatePrometheusMetrics(jsonMetrics)

	// Write the metrics to a file.
	metricsFileLock.Lock()
	defer metricsFileLock.Unlock()

	// Write the metrics into a json file (without ioutil).
	jsonBytes, err := json.Marshal(jsonMetrics)
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

	// Write the metrics into a prometheus file (without ioutil).
	prometheusBytes := []byte(strings.Join(prometheusMetrics, "\n"))
	file, err = os.Create(fmt.Sprintf("%s/metrics.txt", env.StaticPath))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = file.Write(prometheusBytes)
	if err != nil {
		panic(err)
	}
}

// Build the metrics file periodically.
func UpdateMetricsFilesPeriodically() {
	for {
		time.Sleep(1 * time.Second)
		UpdateMetricsFiles()
	}
}
