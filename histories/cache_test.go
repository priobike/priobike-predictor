package histories

import (
	"fmt"
	"predictor/env"
	"predictor/observations"
	"sync"
	"testing"
	"time"
)

func TestHistoryFileConcurrentWriteAndLoad(t *testing.T) {
	var concurrent uint = 0
	tempDir := t.TempDir()
	mockFilePath := fmt.Sprintf("%s/h.json", tempDir)

	var wg sync.WaitGroup
	for {
		if concurrent > 10_000 {
			break
		}
		wg.Add(1)
		go func() {
			appendToHistoryFile(mockFilePath, HistoryCycle{
				StartTime: time.Now(), // Just to have some variation in the writes
			})
			go func() {
				_, err := LoadHistory(mockFilePath)
				if err != nil {
					t.Fail()
				}
				wg.Done()
			}()
		}()
		concurrent += 1
	}
	wg.Wait()
}

func TestBypassCache(t *testing.T) {
	var runs uint = 0
	tempDir := t.TempDir()
	mockFilePath := fmt.Sprintf("%s/h.json", tempDir)

	for {
		if runs > 10_000 {
			break
		}
		appendToHistoryFile(mockFilePath, HistoryCycle{
			StartTime: time.Now(), // Just to have some variation in the writes
		})
		// Cleanup the cache to load from filesystem
		cache.Range(func(key interface{}, _ interface{}) bool {
			cache.Delete(key)
			return true
		})
		_, err := LoadHistory(mockFilePath)
		if err != nil {
			t.Fail()
		}
		runs += 1
	}
}

func TestLoadBestHistory(t *testing.T) {
	tempDir := t.TempDir()
	env.StaticPath = tempDir

	unspecificHistoryFile := fmt.Sprintf("%s/history/%s", tempDir, "1337_1.json")
	specificHistoryFile := fmt.Sprintf("%s/history/%s-P%d.json", tempDir, "1337_1", 123)
	unspecificHistoryCycle1 := HistoryCycle{
		StartTime: time.Now(),
	}
	specificHistoryCycle1 := HistoryCycle{
		StartTime: time.Now(),
	}
	_, err := appendToHistoryFile(unspecificHistoryFile, unspecificHistoryCycle1)
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}
	_, err = appendToHistoryFile(specificHistoryFile, specificHistoryCycle1)
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	// Override the interface to get the current program
	getCurrentProgram = func(_ string) (observations.Observation, bool) {
		mockProgramObservation := observations.Observation{
			Result: 123, // Program ID relevant for selection
		}
		return mockProgramObservation, true
	}

	bestHistory, programId, err := LoadBestFittingHistory("1337_1")
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}
	if programId == nil || *programId != 123 {
		t.Errorf("program id should be 123, but is %x", programId)
		t.FailNow()
	}
	if len(bestHistory.Cycles) != 1 {
		t.Errorf("1 cycle should be stored")
		t.FailNow()
	}
	if bestHistory.Cycles[0].StartTime != specificHistoryCycle1.StartTime {
		t.Errorf("wrong history loaded")
		t.FailNow()
	}
}
