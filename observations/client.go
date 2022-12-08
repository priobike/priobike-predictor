package observations

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"predictor/log"
	"predictor/things"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// The QoS for observations. We use QoS 1 (at least once).
// We also don't use QoS 2 since the messages could be delayed if the
// broker is overloaded or the upstream connection is slow. Note
// that this implies that we might receive the same observation twice.
const observationQoS = 1

// Received messages by their topic.
var receivedMessages = make(map[string]int)

// A lock for receivedMessages.
var receivedMessagesLock = &sync.RWMutex{}

// The number of processed messages, for logging purposes.
var received uint64 = 0
var canceled uint64 = 0
var processed uint64 = 0

// Check out the number of received messages periodically.
func CheckReceivedMessagesPeriodically() {
	for {
		receivedNow := received
		canceledNow := canceled
		processedNow := processed
		time.Sleep(60 * time.Second)
		receivedThen := received
		canceledThen := canceled
		processedThen := processed
		dReceived := receivedThen - receivedNow
		dCanceled := canceledThen - canceledNow
		dProcessed := processedThen - processedNow
		// Panic if the number of received messages is too low.
		if dReceived == 0 {
			panic("No messages received in the last 60 seconds")
		}
		log.Info.Printf("Received %d observations in the last 60 seconds. (%d processed, %d canceled)", dReceived, dProcessed, dCanceled)
		receivedMessagesLock.Lock()
		for dsType, count := range receivedMessages {
			log.Info.Printf("  - Received %d observations for `%s`.", count, dsType)
		}
		receivedMessages = make(map[string]int)
		receivedMessagesLock.Unlock()
	}
}

// Process a message.
func processMessage(msg mqtt.Message) {
	atomic.AddUint64(&received, 1)

	// Add the observation to the correct map.
	topic := msg.Topic()

	// Check if the topic should be processed.
	dsType, ok := things.DatastreamMqttTopics.Load(topic)
	if !ok {
		atomic.AddUint64(&canceled, 1)
		return
	}

	receivedMessagesLock.Lock()
	receivedMessages[dsType.(string)]++
	receivedMessagesLock.Unlock()

	var observation Observation
	if err := json.Unmarshal(msg.Payload(), &observation); err != nil {
		atomic.AddUint64(&canceled, 1)
		return
	}

	err := validateObservation(observation, dsType.(string))
	if err != nil {
		atomic.AddUint64(&canceled, 1)
		log.Warning.Printf("Invalid observation: %s", err)
		return
	}

	switch dsType {
	case "primary_signal":
		thingName, ok := things.PrimarySignalDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ := primarySignalCycles.LoadOrStore(thingName, &Cycle{})
		cycle.(*Cycle).add(observation)
		go PrimarySignalCallback(thingName.(string))
	case "signal_program":
		thingName, ok := things.SignalProgramDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ := signalProgramCycles.LoadOrStore(thingName, &Cycle{})
		cycle.(*Cycle).add(observation)
		go SignalProgramCallback(thingName.(string))
	case "detector_car":
		thingName, ok := things.CarDetectorDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ := carDetectorCycles.LoadOrStore(thingName, &Cycle{})
		cycle.(*Cycle).add(observation)
		go CarDetectorCallback(thingName.(string))
	case "detector_bike":
		thingName, ok := things.BikeDetectorDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ := bikeDetectorCycles.LoadOrStore(thingName, &Cycle{})
		cycle.(*Cycle).add(observation)
		go BikeDetectorCallback(thingName.(string))
	case "cycle_second":
		thingName, ok := things.CycleSecondDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ := cycleSecondCycles.LoadOrStore(thingName, &Cycle{})
		cycle.(*Cycle).add(observation)

		// Make sure that all cycles use the same timeframe.
		newCycleStartTime := cycle.(*Cycle).EndTime
		newCycleEndTime := observation.PhenomenonTime

		// Complete all associated cycles.
		completedCycleSecondCycle, err := cycle.(*Cycle).complete(newCycleStartTime, newCycleEndTime)
		if err != nil {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ = primarySignalCycles.LoadOrStore(thingName, &Cycle{})
		completedPrimarySignalCycle, err := cycle.(*Cycle).complete(newCycleStartTime, newCycleEndTime)
		if err != nil {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ = signalProgramCycles.LoadOrStore(thingName, &Cycle{})
		completedSignalProgramCycle, err := cycle.(*Cycle).complete(newCycleStartTime, newCycleEndTime)
		if err != nil {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ = carDetectorCycles.LoadOrStore(thingName, &Cycle{})
		completedCarDetectorCycle, err := cycle.(*Cycle).complete(newCycleStartTime, newCycleEndTime)
		if err != nil {
			atomic.AddUint64(&canceled, 1)
			return
		}
		cycle, _ = bikeDetectorCycles.LoadOrStore(thingName, &Cycle{})
		completedBikeDetectorCycle, err := cycle.(*Cycle).complete(newCycleStartTime, newCycleEndTime)
		if err != nil {
			atomic.AddUint64(&canceled, 1)
			return
		}

		go CycleSecondCallback(
			thingName.(string),
			newCycleStartTime,
			newCycleEndTime,
			completedPrimarySignalCycle,
			completedSignalProgramCycle,
			completedCycleSecondCycle,
			completedCarDetectorCycle,
			completedBikeDetectorCycle,
		)
	}

	atomic.AddUint64(&processed, 1)
}

// Listen for new observations via mqtt.
func ConnectObservationListener() {
	topics := []string{}
	things.DatastreamMqttTopics.Range(func(topic, _ interface{}) bool {
		topics = append(topics, topic.(string))
		return true
	})

	// Create a new client for every 1000 subscriptions.
	var client mqtt.Client
	var wg sync.WaitGroup
	for i, topic := range topics {
		if (i % 1000) == 0 {
			wg.Wait()
			log.Info.Println("Connecting with new client to observation mqtt broker at :", observationMqttUrl)
			opts := mqtt.NewClientOptions()
			opts.AddBroker(observationMqttUrl)
			opts.SetConnectTimeout(10 * time.Second)
			opts.SetConnectRetry(true)
			opts.SetConnectRetryInterval(5 * time.Second)
			opts.SetAutoReconnect(true)
			opts.SetKeepAlive(60 * time.Second)
			opts.SetPingTimeout(10 * time.Second)
			opts.SetOnConnectHandler(func(client mqtt.Client) {
				log.Info.Println("Connected to observation mqtt broker.")
			})
			opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
				log.Warning.Println("Connection to observation mqtt broker lost:", err)
			})
			randSource := rand.NewSource(time.Now().UnixNano())
			random := rand.New(randSource)
			clientID := fmt.Sprintf("priobike-predictor-%d", random.Int())
			log.Info.Println("Using client id:", clientID)
			opts.SetClientID(clientID)
			opts.SetOrderMatters(false)
			opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
				log.Warning.Println("Received unexpected message on topic:", msg.Topic())
			})
			client = mqtt.NewClient(opts)
			if conn := client.Connect(); conn.Wait() && conn.Error() != nil {
				panic(conn.Error())
			}
		}

		wg.Add(1)
		go func(topic string) {
			defer wg.Done()

			// Subscribe to the datastream.
			if token := client.Subscribe(topic, observationQoS, func(client mqtt.Client, msg mqtt.Message) {
				// Process the message asynchronously to avoid blocking the mqtt client.
				go processMessage(msg)
			}); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
		}(topic)
	}

	log.Info.Println("Subscribed to all datastreams.")
}
