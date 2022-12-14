package observations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"predictor/env"
	"predictor/log"
	"sync"
)

func prefetchMostRecentObservationsPage(page int) (more bool) {
	elementsPerPage := 100
	pageUrl := env.SensorThingsBaseUrl + "Datastreams?" + url.QueryEscape(
		"$filter="+
			"properties/serviceName eq 'HH_STA_traffic_lights' "+
			"and (properties/layerName eq 'signal_program') "+
			"and (Thing/properties/laneType eq 'Radfahrer' "+
			"  or Thing/properties/laneType eq 'KFZ/Radfahrer' "+
			"  or Thing/properties/laneType eq 'Fußgänger/Radfahrer' "+
			"  or Thing/properties/laneType eq 'Bus/Radfahrer' "+
			"  or Thing/properties/laneType eq 'KFZ/Bus/Radfahrer')"+
			"&$expand=Thing,Observations($orderby=phenomenonTime;$top=1)"+
			"&$skip="+fmt.Sprintf("%d", page*elementsPerPage),
	)
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
		o := expandedDatastream.Observations[0]
		switch expandedDatastream.Properties.LayerName {
		// At the moment, we only care about signal programs.
		case "signal_program":
			cycle, _ := signalProgramCycles.LoadOrStore(expandedDatastream.Thing.Name, &Cycle{})
			cycle.(*Cycle).add(o)
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
		log.Info.Printf("Bulk prefetching observations from pages %d-%d...", page-10, page-1)
		wg.Wait()
		if !foundMore {
			break
		}
	}

	log.Info.Println("Prefetched most recent observations.")
}
