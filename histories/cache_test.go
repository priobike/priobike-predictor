package histories

import (
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

func TestHistoryFileConcurrentWrite(t *testing.T) {
	var concurrentWrites uint = 0
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf(err.Error())
	}
	mockFilePath := fmt.Sprintf("%s/h.json", tempDir)

	var wg sync.WaitGroup
	for {
		if concurrentWrites > 10_000 {
			break
		}
		wg.Add(1)
		go func() {
			appendToHistoryFile(mockFilePath, HistoryCycle{
				StartTime: time.Now(), // Just to have some variation in the writes
			})
			wg.Done()
		}()
		concurrentWrites += 1
	}
	wg.Wait()
}
