package histories

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/env"
	"sync"
	"time"
)

type IndexEntry struct {
	// The json file.
	File string `json:"file"`
	// The time when the history was last updated.
	LastUpdated time.Time `json:"lastUpdated"`
	// If a car was detected in the history.
	CarDetected bool `json:"carDetected"`
	// If a bike was detected in the history.
	BikeDetected bool `json:"bikeDetected"`
	// The number of cycles in the history.
	CycleCount int `json:"cycleCount"`
}

// The lock that must be used when writing or reading the index file.
// This is to gobally protect concurrent access to the same file.
var indexFileLock = &sync.Mutex{}

// Lookup all .json history files in the static path and write them into a json file.
// This serves as an index for the cycle analyzer.
func UpdateHistoryIndex() {
	entries := make([]IndexEntry, 0)
	cache.Range(func(key, value interface{}) bool {
		path := key.(string)
		history := value.(History)

		var carDetected = false
		var bikeDetected = false
		var cycleLenSum = 0
		for _, cycle := range history.Cycles {
			if len(cycle.Cars) > 0 {
				carDetected = true
			}
			if len(cycle.Bikes) > 0 {
				bikeDetected = true
			}
			cycleLenSum += len(cycle.Phases)
		}
		var cycleCount = len(history.Cycles)
		if cycleCount == 0 {
			return true
		}
		// Split the filename from the path.
		var filename = path
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' {
				filename = path[i+1:]
				break
			}
		}
		// Add the history to the index.
		entries = append(entries, IndexEntry{
			File:         filename,
			LastUpdated:  history.Cycles[cycleCount-1].EndTime,
			CarDetected:  carDetected,
			BikeDetected: bikeDetected,
			CycleCount:   cycleCount,
		})

		return true
	})

	// Write the json files into a json file (without ioutil).
	jsonBytes, err := json.Marshal(entries)
	if err != nil {
		panic(err)
	}
	// Acquire the file locks for additional safety.
	indexFileLock.Lock()
	defer indexFileLock.Unlock()
	indexFile, err := os.Create(fmt.Sprintf("%s/index.json", env.StaticPath))
	if err != nil {
		panic(err)
	}
	defer indexFile.Close()
	_, err = indexFile.Write(jsonBytes)
	if err != nil {
		panic(err)
	}
}

// Build the index file periodically.
func UpdateHistoryIndexPeriodically() {
	for {
		time.Sleep(10 * time.Second)
		UpdateHistoryIndex()
	}
}
