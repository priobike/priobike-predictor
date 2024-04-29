package histories

import (
	"encoding/json"
	"fmt"
	"os"
	"predictor/env"
	"reflect"
	"testing"
	"time"
)

func TestHistoryIndex(t *testing.T) {
	// Cleanup the history
	cache.Range(func(key interface{}, value interface{}) bool {
		cache.Delete(key)
		return true
	})
	// Write some dummy histories in the cache.
	tempDir := t.TempDir()
	exampleCycle := HistoryCycle{
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(10, 0), // Usually cycles are longer, but suffices for testing
		Phases: []HistoryPhaseEvent{
			// Green at 0 seconds
			{
				Time:  time.Unix(0, 0),
				Color: 3,
			},
			// Red at 5 seconds
			{
				Time:  time.Unix(5, 0),
				Color: 1,
			},
		},
	}
	history := History{
		Cycles: []HistoryCycle{
			exampleCycle,
			exampleCycle,
		},
	}
	jsonData, err := json.Marshal(history)
	if err != nil {
		t.Errorf("failed to marshal json data: %s", err.Error())
		t.FailNow()
	}
	historyFilePath := fmt.Sprintf("%s/1337_1.json", tempDir)
	historyFile, err := os.Create(historyFilePath)
	if err != nil {
		t.Errorf("history file could not be created: %s", err.Error())
		t.FailNow()
	}
	defer historyFile.Close()
	_, err = historyFile.Write(jsonData)
	if err != nil {
		t.Errorf("could not write into history file: %s", err.Error())
		t.FailNow()
	}
	cache.Store(historyFilePath, history)

	env.StaticPath = tempDir
	UpdateHistoryIndex()

	indexFilePath := fmt.Sprintf("%s/index.json", tempDir)
	indexFile, err := os.OpenFile(indexFilePath, os.O_RDONLY, 0644)
	if err != nil {
		t.Errorf("could not open index file: %s", err.Error())
		t.FailNow()
	}
	defer indexFile.Close()

	decoder := json.NewDecoder(indexFile)
	var unmarshaledIndex []IndexEntry
	err = decoder.Decode(&unmarshaledIndex)
	if err != nil {
		t.Errorf("could not unmarshal data from index file: %s", err.Error())
		t.FailNow()
	}

	expectedIndex := []IndexEntry{
		{
			File:        "1337_1.json",
			LastUpdated: time.Unix(10, 0),
			CycleCount:  2,
		},
	}

	// Using deep equals here is fine
	if !reflect.DeepEqual(unmarshaledIndex, expectedIndex) {
		t.Errorf("expected index does not correspond with unmarshaled index: %v != %v", expectedIndex, unmarshaledIndex)
	}
}
