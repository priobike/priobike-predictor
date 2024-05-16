package predictions

import (
	"fmt"
	"math"
	"predictor/observations"
	"predictor/things"
	"sync"
	"time"
)

var (
	predictedStates     = &sync.Map{}
	actualStates        = &sync.Map{}
	predictionQualities = &sync.Map{}
)

func GetPredictionQualities() map[string]float64 {
	qualities := map[string]float64{}
	predictionQualities.Range(func(k, v interface{}) bool {
		qualities[k.(string)] = v.(float64)
		return true
	})
	return qualities
}

func calculatePredictionQuality(thingName string) error {
	// Get the current state of the thing.
	primarySignalObservation, ok := observations.GetCurrentPrimarySignal(thingName)
	if !ok {
		return fmt.Errorf("no primary signal observation available")
	}
	timeDelay := primarySignalObservation.ReceivedTime.
		Sub(primarySignalObservation.PhenomenonTime)

	// If the time delay is too large (>10min), store -1 as quality.
	if timeDelay > 10*time.Minute {
		predictionQualities.Store(thingName, float64(-1))
		return nil
	}

	nowWithDelay := time.Now().Add(-timeDelay)

	prediction, ok := GetCurrentPrediction(thingName)
	if !ok {
		return fmt.Errorf("no prediction available")
	}

	delayedTimeInPrediction := int(math.Abs(
		nowWithDelay.Sub(prediction.ReferenceTime).Seconds(),
	))
	lenNow := len(prediction.Now)
	lenThen := len(prediction.Then)
	var predictedColor byte
	if delayedTimeInPrediction < len(prediction.Now) {
		i := delayedTimeInPrediction % lenNow
		predictedColor = prediction.Now[i]
	} else {
		i := (delayedTimeInPrediction - lenNow) % lenThen
		predictedColor = prediction.Then[i]
	}

	actualColor := primarySignalObservation.Result

	predictedStatesValue, _ := predictedStates.LoadOrStore(thingName, []byte{})
	actualStatesValue, _ := actualStates.LoadOrStore(thingName, []byte{})
	predictedStatesArr := predictedStatesValue.([]byte)
	actualStatesArr := actualStatesValue.([]byte)
	predictedStatesArr = append(predictedStatesArr, predictedColor)
	actualStatesArr = append(actualStatesArr, actualColor)
	// Remove the first element if the array is too long.
	if len(predictedStatesArr) > 120 {
		predictedStatesArr = predictedStatesArr[1:]
	}
	if len(actualStatesArr) > 120 {
		actualStatesArr = actualStatesArr[1:]
	}
	predictedStates.Store(thingName, predictedStatesArr)
	actualStates.Store(thingName, actualStatesArr)

	// Calculate the quality of the prediction.
	var correct int
	if len(predictedStatesArr) != len(actualStatesArr) {
		return fmt.Errorf("invalid state length")
	}
	for i := 0; i < len(predictedStatesArr); i++ {
		if predictedStatesArr[i] == actualStatesArr[i] {
			correct++
		}
	}

	quality := float64(correct) / float64(len(predictedStatesArr)) // Never 0 elements
	predictionQualities.Store(thingName, quality)

	return nil
}

func CheckPredictionQualityPeriodically() {
	for {
		things.Things.Range(func(k, v interface{}) bool {
			err := calculatePredictionQuality(k.(string))
			if err != nil {
				fmt.Printf("Error calculating prediction quality: %v\n", err)
			}
			return true
		})
		time.Sleep(1 * time.Second)
	}
}
