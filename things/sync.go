package things

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"predictor/log"
	"sync"
)

// A map that contains all things by their name.
var Things = &sync.Map{}

// A map that contains all thing names by their crossing id.
var Crossings = &sync.Map{}

// A map that contains all datastream MQTT topics to subscribe to, by their type.
var DatastreamMqttTopics = &sync.Map{}

// A map that points `primary_signal` Datastream MQTT topics to Thing names.
var PrimarySignalDatastreams = &sync.Map{}

// A map that points `signal_program` Datastream MQTT topics to Thing names.
var SignalProgramDatastreams = &sync.Map{}

// A map that points `cycle_second` Datastream MQTT topics to Thing names.
var CycleSecondDatastreams = &sync.Map{}

// A map that points `detector_car` Datastream MQTT topics to Thing names.
var CarDetectorDatastreams = &sync.Map{}

// A map that points `detector_bike` Datastream MQTT topics to Thing names.
var BikeDetectorDatastreams = &sync.Map{}

func syncThingsPage(page int) (more bool) {
	elementsPerPage := 100
	queryUrl := baseUrl + "Things?%24filter=" + url.QueryEscape(thingsQuery)
	pageUrl := queryUrl + "&%24skip=" + url.QueryEscape(fmt.Sprintf("%d", page*elementsPerPage))

	resp, err := http.Get(pageUrl)
	if err != nil {
		log.Warning.Println("Could not sync things:", err)
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warning.Println("Could not sync things:", err)
		panic(err)
	}

	var thingsResponse struct {
		Value   []Thing `json:"value"`
		NextUri *string `json:"@iot.nextLink"`
	}
	if err := json.Unmarshal(body, &thingsResponse); err != nil {
		log.Warning.Println("Could not sync things:", err)
		panic(err)
	}

	for _, t := range thingsResponse.Value {
		// Add the thing to the things map.
		Things.Store(t.Name, t)

		// Add the thing name to the crossing map.
		cs, _ := Crossings.LoadOrStore(t.Properties.TrafficLightsId, []string{})
		cs = append(cs.([]string), t.Name)
		Crossings.Store(t.Properties.TrafficLightsId, cs)

		for _, d := range t.Datastreams {
			switch d.Properties.LayerName {
			case "primary_signal":
				DatastreamMqttTopics.Store(d.MqttTopic(), "primary_signal")
				PrimarySignalDatastreams.Store(d.MqttTopic(), t.Name)
			case "signal_program":
				DatastreamMqttTopics.Store(d.MqttTopic(), "signal_program")
				SignalProgramDatastreams.Store(d.MqttTopic(), t.Name)
			case "cycle_second":
				DatastreamMqttTopics.Store(d.MqttTopic(), "cycle_second")
				CycleSecondDatastreams.Store(d.MqttTopic(), t.Name)
			case "detector_car":
				DatastreamMqttTopics.Store(d.MqttTopic(), "detector_car")
				CarDetectorDatastreams.Store(d.MqttTopic(), t.Name)
			case "detector_bike":
				DatastreamMqttTopics.Store(d.MqttTopic(), "detector_bike")
				BikeDetectorDatastreams.Store(d.MqttTopic(), t.Name)
			}
		}
	}

	return thingsResponse.NextUri != nil
}

// Periodically sync the things from the SensorThings API.
func SyncThings() {
	log.Info.Println("Syncing things...")

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
				more := syncThingsPage(page)
				if more {
					foundMore = true
				}
			}(page)
			page++
		}
		log.Info.Printf("Bulk syncing things from %s pages %d-%d...", baseUrl, page-10, page-1)
		wg.Wait()
		if !foundMore {
			break
		}
	}

	log.Info.Println("Synced things.")
}
