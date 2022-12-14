package histories

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/env"
	"predictor/log"
	"predictor/observations"
	"sync"
	"sync/atomic"
)

// The maximum length of the history files.
// A longer history will be more robust for statistical evaluation.
// A shorter history will react faster to changes in the program behavior.
const maxHistoryLength = 10

// The current histories by their file path.
// The cache is used to speedup access to the history files.
var cache = sync.Map{}

// Locks that must be used when writing or reading a history file.
var historyFileLocks = &sync.Map{}

// Append a new cycle to a new or existing history file.
func appendToHistoryFile(path string, newCycle HistoryCycle) (History, error) {
	// Unmarshal the history from the file, if it exists.
	// If none exists, create a new history.
	val, _ := cache.LoadOrStore(path, History{})
	history := val.(History)
	// Append the new cycle to the history.
	history.Cycles = append(history.Cycles, newCycle)
	// If the history is too long, remove the oldest cycles.
	if len(history.Cycles) > maxHistoryLength {
		history.Cycles = history.Cycles[len(history.Cycles)-maxHistoryLength:]
	}
	// Acquire the file lock for additional safety.
	lock, _ := historyFileLocks.LoadOrStore(path, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()
	// Marshal the history to the file.
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Error.Println(err)
		atomic.AddUint64(&canceled, 1)
		return History{}, err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(history)
	if err != nil {
		log.Error.Println(err)
		atomic.AddUint64(&canceled, 1)
		return History{}, err
	}

	cache.Store(path, history)
	return history, nil
}

// Load a history from a file path (or directly from the cache).
func LoadHistory(path string) (History, error) {
	historyFromCache, ok := cache.Load(path)
	if ok {
		return historyFromCache.(History), nil
	}
	// Load the history from the file and populate the cache.
	lock, _ := historyFileLocks.LoadOrStore(path, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()
	// Load the history from the file.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return History{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var historyFromFile History
	err = decoder.Decode(&historyFromFile)
	if err != nil {
		return History{}, err
	}
	// Populate the cache.
	cache.Store(path, historyFromFile)
	return historyFromFile, nil
}

// Load the best fitting history for a given thing name.
// This will lookup the program currently running on the thing and load the corresponding history.
// If no such history exists, it will load the default history.
func LoadBestFittingHistory(thingName string) (history History, programId *byte, err error) {
	programsToSearch := []*byte{}
	// Lookup the last running program.
	if programObservation, ok := observations.GetCurrentProgram(thingName); ok {
		programsToSearch = append(programsToSearch, &programObservation.Result)
	}
	programsToSearch = append(programsToSearch, nil)
	for _, programId := range programsToSearch {
		var path = fmt.Sprintf("%s/history/%s.json", env.StaticPath, thingName)
		if programId != nil {
			path = fmt.Sprintf("%s/history/%s-P%d.json", env.StaticPath, thingName, *programId)
		}
		history, err := LoadHistory(path)
		if err != nil {
			continue
		}
		return history, programId, nil
	}
	return History{}, nil, fmt.Errorf("no history found for thing %s", thingName)
}
