package observations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"predictor/log"
	"sync"
)

func prefetchMostRecentObservationsPage(page int) (more bool) {
	elementsPerPage := 100
	queryUrl := baseUrl + "Datastreams?%24filter=" + url.QueryEscape(observationsQuery)
	pageUrl := queryUrl + "&%24skip=" + url.QueryEscape(fmt.Sprintf("%d", page*elementsPerPage))
	resp, err := http.Get(pageUrl)
	if err != nil {
		log.Warning.Println("Could not sync observations:", err)
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warning.Println("Could not sync observations:", err)
		panic(err)
	}

	var observationsResponse struct {
		Value []struct {
			DatastreamId int `json:"@iot.id"`
			Properties   struct {
				LayerName string `json:"layerName"`
			}
			Thing struct {
				Name string `json:"name"`
			}
			Observations []Observation `json:"Observations"`
		} `json:"value"`
		NextUri *string `json:"@iot.nextLink"`
	}
	if err := json.Unmarshal(body, &observationsResponse); err != nil {
		log.Warning.Println("Could not sync observations:", err)
		panic(err)
	}

	for _, expandedDatastream := range observationsResponse.Value {
		if len(expandedDatastream.Observations) == 0 {
			continue
		}
		switch expandedDatastream.Properties.LayerName {
		case "primary_signal":
			cycle, _ := primarySignalCycles.LoadOrStore(expandedDatastream.Thing.Name, &Cycle{})
			cycle.(*Cycle).add(expandedDatastream.Observations[0])
		case "signal_program":
			cycle, _ := signalProgramCycles.LoadOrStore(expandedDatastream.Thing.Name, &Cycle{})
			cycle.(*Cycle).add(expandedDatastream.Observations[0])
		case "detector_car":
			cycle, _ := carDetectorCycles.LoadOrStore(expandedDatastream.Thing.Name, &Cycle{})
			cycle.(*Cycle).add(expandedDatastream.Observations[0])
		case "detector_bike":
			cycle, _ := bikeDetectorCycles.LoadOrStore(expandedDatastream.Thing.Name, &Cycle{})
			cycle.(*Cycle).add(expandedDatastream.Observations[0])
		case "cycle_second":
			cycle, _ := cycleSecondCycles.LoadOrStore(expandedDatastream.Thing.Name, &Cycle{})
			cycle.(*Cycle).add(expandedDatastream.Observations[0])
		default:
			continue
		}
	}
	return observationsResponse.NextUri != nil
}

// Prefetch the most recent observations for all datastreams.
func PrefetchMostRecentObservations() {
	log.Info.Println("Prefetching most recent observations...")

	// Fetch all pages of the SensorThings query.
	var page = 0
	for {
		// Make some parallel requests to speed things up.
		var wg sync.WaitGroup
		var foundMore = false
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(page int) {
				defer wg.Done()
				more := prefetchMostRecentObservationsPage(page)
				if more {
					foundMore = true
				}
			}(page)
			page++
		}
		log.Info.Printf("Bulk prefetching observations from %s pages %d-%d...", baseUrl, page-10, page-1)
		wg.Wait()
		if !foundMore {
			break
		}
	}

	log.Info.Println("Prefetched most recent observations.")
}
