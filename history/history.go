package history

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/log"
	"predictor/observations"
	"predictor/things"
	"sort"
	"sync"
)

// A lock for the filesystem.
var fileLock = &sync.Mutex{}

// Build the history for the given thing.
func Build(thingName string) {
	// Get the signal colors in the last cycle.
	primarySignalCycle, ok := observations.GetPrimarySignalCycle(thingName)
	if !ok {
		return
	}
	// Check if the time frame of the cycle is valid.
	// The time difference should be less than 200 seconds.
	if primarySignalCycle.EndTime.Unix()-primarySignalCycle.StartTime.Unix() > 200 {
		return
	}
	// Check if the cycle is empty.
	if len(primarySignalCycle.Completed) == 0 {
		return
	}

	// Get the last program that was running on the signal.
	signalProgramCycle, ok := observations.GetSignalProgramCycle(thingName)
	if !ok {
		return
	}
	// We don't want a cycle where the program changed inbetween.
	if len(signalProgramCycle.FindObservationsInRange(
		primarySignalCycle.StartTime,
		primarySignalCycle.EndTime,
	)) > 0 {
		return
	}
	var programId *int
	mostRecentSignalProgramObservation, err := signalProgramCycle.GetMostRecentObservation()
	if err == nil {
		programId = &mostRecentSignalProgramObservation.Result
	}

	// Get the thing and all other traffic lights on the same crossing.
	thing, ok := things.GetThing(thingName)
	if !ok {
		log.Error.Printf("Could not find thing with name " + thingName)
		return
	}
	thingsNamesAtCrossing, ok := things.GetCrossing(thing.CrossingId())
	if !ok {
		log.Error.Printf("Could not find things at crossing with id " + thing.CrossingId())
		return
	}

	// Check the car detector state for each thing at the crossing.
	var detectedCars = make(map[string]bool)
	for _, thingNameAtCrossing := range thingsNamesAtCrossing {
		carDetectorCycle, ok := observations.GetCarDetectorCycle(thingName)
		if !ok {
			continue
		}
		// Check if there was any car detected in this cycle.
		os := carDetectorCycle.FindObservationsInRange(
			primarySignalCycle.StartTime,
			primarySignalCycle.EndTime,
		)
		for _, o := range os {
			if o.Result > 0 {
				detectedCars[thingNameAtCrossing] = true
				break
			}
		}
	}

	// Check the bike detector state for each thing at the crossing.
	var detectedBikes = make(map[string]bool)
	for _, thingNameAtCrossing := range thingsNamesAtCrossing {
		bikeDetectorCycle, ok := observations.GetBikeDetectorCycle(thingName)
		if !ok {
			continue
		}
		// Check if there was any bike detected in this cycle.
		os := bikeDetectorCycle.FindObservationsInRange(
			primarySignalCycle.StartTime,
			primarySignalCycle.EndTime,
		)
		for _, o := range os {
			if o.Result > 0 {
				detectedBikes[thingNameAtCrossing] = true
				break
			}
		}
	}

	// Reconstruct the signal values in the cycle, from StartTime to EndTime.
	values := make([]int, 0)
	for second := primarySignalCycle.StartTime.Unix(); second <= primarySignalCycle.EndTime.Unix(); second++ {
		// Get the signal value for the current second.
		o, err := primarySignalCycle.FindObservationForSecond(second)
		if err != nil {
			return // Don't build the history if the cycle is incomplete.
		}
		values = append(values, o.Result)
	}

	// Append this history to the history file.
	// This file will be named after the following scheme:
	// <staticPath>/<thingName>.json
	path := fmt.Sprintf("%s/%s", staticPath, thingName)
	// If we know the running program, we will append the program id to the file name:
	// <staticPath>/<thingName>-p<programId>.json
	if programId != nil {
		path += fmt.Sprintf("-p%d", *programId)
	}
	// If cars were detected, we will append the car detector thing names to the file name:
	// <staticPath>/<thingName>-c<carThingName>.json
	if len(detectedCars) > 0 {
		// Sort the car thing names so that the file name is always the same.
		carThingNames := make([]string, 0)
		for carThingName := range detectedCars {
			carThingNames = append(carThingNames, carThingName)
		}
		sort.Strings(carThingNames)
		for _, carThingName := range carThingNames {
			path += fmt.Sprintf("-c%s", carThingName)
		}
	}
	// If bikes were detected, we will append the bike detector thing names to the file name:
	// <staticPath>/<thingName>-b<bikeThingName>.json
	if len(detectedBikes) > 0 {
		// Sort the bike thing names so that the file name is always the same.
		bikeThingNames := make([]string, 0)
		for bikeThingName := range detectedBikes {
			bikeThingNames = append(bikeThingNames, bikeThingName)
		}
		sort.Strings(bikeThingNames)
		for _, bikeThingName := range bikeThingNames {
			path += fmt.Sprintf("-b%s", bikeThingName)
		}
	}
	path += ".json"

	fileLock.Lock()
	defer fileLock.Unlock()

	// Unmarshal the history from the file, if it exists.
	// If none exists, create a new history.
	history := [][]int{}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Error.Println(err)
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.Decode(&history)
	// Append the new cycle to the history.
	history = append(history, values)
	// If the history is too long, remove the oldest cycles.
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	// Marshal the history to the file, with pretty printing.
	file.Truncate(0)
	file.Seek(0, 0)
	// For each cycle, make a newline.
	file.WriteString("[")
	for i, values := range history {
		file.WriteString("\n")
		file.WriteString("  [")
		for j, v := range values {
			file.WriteString(fmt.Sprintf("%d", v))
			if j < len(values)-1 {
				file.WriteString(",")
			}
		}
		file.WriteString("]")
		if i < len(history)-1 {
			file.WriteString(",")
		}
	}
	file.WriteString("\n]")
}
