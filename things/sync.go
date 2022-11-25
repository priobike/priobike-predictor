package things

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"predictor/env"
	"predictor/log"
	"sync"
	"time"
)

// The SensorThings API base URL.
var baseUrl = env.Load("SENSORTHINGS_URL")

// The SensorThings query used to fetch the Things.
var query = env.Load("SENSORTHINGS_QUERY")

// A map that contains all things by their name.
var things = make(map[string]Thing)

// A lock for the things map.
var thingsMutex = &sync.Mutex{}

// Get a thing by its name.
func GetThing(name string) (Thing, bool) {
	thingsMutex.Lock()
	thing, ok := things[name]
	thingsMutex.Unlock()
	return thing, ok
}

// A map that contains all thing names by their crossing id.
var crossings = make(map[string][]string)

// A lock for the crossings map.
var crossingsMutex = &sync.Mutex{}

// Get the Thing names for a crossing id.
func GetCrossing(crossingId string) ([]string, bool) {
	crossingsMutex.Lock()
	thingNames, ok := crossings[crossingId]
	crossingsMutex.Unlock()
	return thingNames, ok
}

// A map that points `primary_signal` Datastream MQTT topics to Thing names.
var primarySignals = make(map[string]string)

// A lock for the primary signals map.
var primarySignalsMutex = &sync.Mutex{}

// Get the Thing name for a `primary_signal` Datastream MQTT topic.
func GetPrimarySignal(topic string) (string, bool) {
	primarySignalsMutex.Lock()
	thingName, ok := primarySignals[topic]
	primarySignalsMutex.Unlock()
	return thingName, ok
}

// A map that points `signal_program` Datastream MQTT topics to Thing names.
var signalPrograms = make(map[string]string)

// A lock for the signal programs map.
var signalProgramsMutex = &sync.Mutex{}

// Get the Thing name for a `signal_program` Datastream MQTT topic.
func GetSignalProgram(topic string) (string, bool) {
	signalProgramsMutex.Lock()
	thingName, ok := signalPrograms[topic]
	signalProgramsMutex.Unlock()
	return thingName, ok
}

// A map that points `detector_car` Datastream MQTT topics to Thing names.
var carDetectors = make(map[string]string)

// A lock for the car detectors map.
var carDetectorsMutex = &sync.Mutex{}

// Get the Thing name for a `detector_car` Datastream MQTT topic.
func GetCarDetector(topic string) (string, bool) {
	carDetectorsMutex.Lock()
	thingName, ok := carDetectors[topic]
	carDetectorsMutex.Unlock()
	return thingName, ok
}

// A map that points `detector_bike` Datastream MQTT topics to Thing names.
var bikeDetectors = make(map[string]string)

// A lock for the bike detectors map.
var bikeDetectorsMutex = &sync.Mutex{}

// Get the Thing name for a `detector_bike` Datastream MQTT topic.
func GetBikeDetector(topic string) (string, bool) {
	bikeDetectorsMutex.Lock()
	thingName, ok := bikeDetectors[topic]
	bikeDetectorsMutex.Unlock()
	return thingName, ok
}

// A map that points `cycle_second` Datastream MQTT topics to Thing names.
var cycleSeconds = make(map[string]string)

// A lock for the cycle seconds map.
var cycleSecondsMutex = &sync.Mutex{}

// Get the Thing name for a `cycle_second` Datastream MQTT topic.
func GetCycleSecond(topic string) (string, bool) {
	cycleSecondsMutex.Lock()
	thingName, ok := cycleSeconds[topic]
	cycleSecondsMutex.Unlock()
	return thingName, ok
}

// Periodically sync the things from the SensorThings API.
func Sync() {
	for {
		log.Info.Println("Syncing things...")

		// Fetch all pages of the SensorThings query.
		var pageUrl = baseUrl + "Things?%24filter=" + url.QueryEscape(query)
		for {
			resp, err := http.Get(pageUrl)
			if err != nil {
				log.Warning.Println("Could not sync things:", err)
				break
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Warning.Println("Could not sync things:", err)
				break
			}

			var thingsResponse ThingsResponse
			if err := json.Unmarshal(body, &thingsResponse); err != nil {
				log.Warning.Println("Could not sync things:", err)
				break
			}

			for _, thing := range thingsResponse.Value {
				thingsMutex.Lock()
				things[thing.Name] = thing
				thingsMutex.Unlock()

				crossingsMutex.Lock()
				crossings[thing.CrossingId()] = append(crossings[thing.CrossingId()], thing.Name)
				crossingsMutex.Unlock()

				for _, datastream := range thing.Datastreams {
					switch datastream.Properties.LayerName {
					case "primary_signal":
						primarySignalsMutex.Lock()
						primarySignals[datastream.MqttTopic()] = thing.Name
						primarySignalsMutex.Unlock()
					case "signal_program":
						signalProgramsMutex.Lock()
						signalPrograms[datastream.MqttTopic()] = thing.Name
						signalProgramsMutex.Unlock()
					case "cycle_second":
						cycleSecondsMutex.Lock()
						cycleSeconds[datastream.MqttTopic()] = thing.Name
						cycleSecondsMutex.Unlock()
					case "detector_car":
						carDetectorsMutex.Lock()
						carDetectors[datastream.MqttTopic()] = thing.Name
						carDetectorsMutex.Unlock()
					case "detector_bike":
						bikeDetectorsMutex.Lock()
						bikeDetectors[datastream.MqttTopic()] = thing.Name
						bikeDetectorsMutex.Unlock()
					}
				}
			}

			if thingsResponse.NextUri == nil {
				break
			}
			pageUrl = *thingsResponse.NextUri
		}

		log.Info.Printf("Synced %d things", len(things))
		log.Info.Printf("Synced %d crossings", len(crossings))
		log.Info.Printf("Synced %d primary_signal datastreams", len(primarySignals))
		log.Info.Printf("Synced %d signal_program datastreams", len(signalPrograms))
		log.Info.Printf("Synced %d cycle_second datastreams", len(cycleSeconds))
		log.Info.Printf("Synced %d detector_car datastreams", len(carDetectors))
		log.Info.Printf("Synced %d detector_bike datastreams", len(bikeDetectors))

		// Sleep for 1 hour.
		time.Sleep(1 * time.Hour)
	}
}
