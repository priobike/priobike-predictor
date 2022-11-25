package observations

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"predictor/env"
	"predictor/log"
	"sync"
	"time"

	"predictor/things"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttUrl = env.Load("MQTT_URL")

// The delegate that is called when a cycle *was* flushed.
var delegate func(thingName string)

// A map that contains all `primary_signal` cycles to the Thing name.
// Primary signal observations tell which "color" the traffic light is currently showing.
var primarySignalCycles = make(map[string]Cycle)

// A lock for the primary signal cycles.
var primarySignalLock sync.Mutex

// Get the `primary_signal` window for a Thing name.
func GetPrimarySignalCycle(thingName string) (Cycle, bool) {
	primarySignalLock.Lock()
	defer primarySignalLock.Unlock()
	window, ok := primarySignalCycles[thingName]
	return window, ok
}

// A map that contains all received `signal_program` observations to the Thing name.
// Signal program observations tell which program the traffic light is currently running.
var signalProgramCycles = make(map[string]Cycle)

// A lock for the signal program cycles.
var signalProgramLock sync.Mutex

// Get the `signal_program` window for a Thing name.
func GetSignalProgramCycle(thingName string) (Cycle, bool) {
	signalProgramLock.Lock()
	defer signalProgramLock.Unlock()
	window, ok := signalProgramCycles[thingName]
	return window, ok
}

// A map that contains all received `cycle_second` observations to the Thing name.
// Cycle second observations tell when a new cycle starts.
var cycleSecondCycles = make(map[string]Cycle)

// A lock for the cycle second cycles.
var cycleSecondLock sync.Mutex

// Get the `cycle_second` window for a Thing name.
func GetCycleSecondCycle(thingName string) (Cycle, bool) {
	cycleSecondLock.Lock()
	defer cycleSecondLock.Unlock()
	window, ok := cycleSecondCycles[thingName]
	return window, ok
}

// A map that contains all received `detector_car` observations to the Thing name.
// Detector car observations tell when a car is detected, from 0 to 100 pct.
var carDetectorCycles = make(map[string]Cycle)

// A lock for the car detector cycles.
var carDetectorLock sync.Mutex

// Get the `detector_car` window for a Thing name.
func GetCarDetectorCycle(thingName string) (Cycle, bool) {
	carDetectorLock.Lock()
	defer carDetectorLock.Unlock()
	window, ok := carDetectorCycles[thingName]
	return window, ok
}

// A map that contains all received `detector_bike` observations to the Thing name.
// Detector bike observations tell when a bike is detected, from 0 to 100 pct.
var bikeDetectorCycles = make(map[string]Cycle)

// A lock for the bike detector cycles.
var bikeDetectorLock sync.Mutex

// Get the `detector_bike` window for a Thing name.
func GetBikeDetectorCycle(thingName string) (Cycle, bool) {
	bikeDetectorLock.Lock()
	defer bikeDetectorLock.Unlock()
	window, ok := bikeDetectorCycles[thingName]
	return window, ok
}

// The number of received messages, for logging purposes.
var received int

// A callback that is executed when new messages arrive on the mqtt topic.
func onMessageReceived(client mqtt.Client, msg mqtt.Message) {
	// Parse the message.
	var observation Observation
	if err := json.Unmarshal(msg.Payload(), &observation); err != nil {
		// Some other observation that we don't care about.
		return
	}

	// Add the observation to the correct map.
	topic := msg.Topic()
	if thingName, ok := things.GetPrimarySignal(topic); ok {
		primarySignalLock.Lock()
		window := primarySignalCycles[thingName]
		window.add(observation)
		primarySignalCycles[thingName] = window
		primarySignalLock.Unlock()
	} else if thingName, ok := things.GetSignalProgram(topic); ok {
		signalProgramLock.Lock()
		window := signalProgramCycles[thingName]
		window.add(observation)
		signalProgramCycles[thingName] = window
		signalProgramLock.Unlock()
	} else if thingName, ok := things.GetCarDetector(topic); ok {
		carDetectorLock.Lock()
		window := carDetectorCycles[thingName]
		window.add(observation)
		carDetectorCycles[thingName] = window
		carDetectorLock.Unlock()
	} else if thingName, ok := things.GetBikeDetector(topic); ok {
		bikeDetectorLock.Lock()
		window := bikeDetectorCycles[thingName]
		window.add(observation)
		bikeDetectorCycles[thingName] = window
		bikeDetectorLock.Unlock()
	} else if thingName, ok := things.GetCycleSecond(topic); ok {
		cycleSecondLock.Lock()
		window := cycleSecondCycles[thingName]
		window.add(observation)
		cycleSecondCycles[thingName] = window
		cycleSecondLock.Unlock()

		onCycleCompleted(thingName, observation.PhenomenonTime)
	} else {
		return
	}
	// Increment the number of received messages.
	received++
}

// A callback that is executed when a full cycle is completed.
// This will flush the observations except for the most recent one.
func onCycleCompleted(thingName string, cycleCompletionTime time.Time) {
	// Acquire the locks.
	primarySignalLock.Lock()
	signalProgramLock.Lock()
	cycleSecondLock.Lock()
	carDetectorLock.Lock()
	bikeDetectorLock.Lock()

	// Flush the observations.
	window := primarySignalCycles[thingName]
	window.complete(cycleCompletionTime)
	primarySignalCycles[thingName] = window

	window = signalProgramCycles[thingName]
	window.complete(cycleCompletionTime)
	signalProgramCycles[thingName] = window

	window = carDetectorCycles[thingName]
	window.complete(cycleCompletionTime)
	carDetectorCycles[thingName] = window

	window = bikeDetectorCycles[thingName]
	window.complete(cycleCompletionTime)
	bikeDetectorCycles[thingName] = window

	// Release the locks.
	primarySignalLock.Unlock()
	signalProgramLock.Unlock()
	cycleSecondLock.Unlock()
	carDetectorLock.Unlock()
	bikeDetectorLock.Unlock()

	// Tell the delegate that a cycle has completed.
	go delegate(thingName)
}

// Print out the number of received messages periodically.
func Print() {
	for {
		time.Sleep(5 * time.Second)
		log.Info.Printf("Received %d observations since service startup.", received)
	}
}

// Listen for new observations via mqtt.
func Listen(onCycleCompleted func(thingName string)) {
	delegate = onCycleCompleted

	log.Info.Println("Connecting to prediction mqtt broker at :", mqttUrl)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Info.Println("Connected to prediction mqtt broker.")
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Warning.Println("Connection to prediction mqtt broker lost:", err)
	})
	randSource := rand.NewSource(time.Now().UnixNano())
	random := rand.New(randSource)
	clientID := fmt.Sprintf("priobike-prediction-service-%d", random.Int())
	log.Info.Println("Using client id:", clientID)
	opts.SetClientID(clientID)
	opts.SetOrderMatters(false)
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Warning.Println("Received unexpected message on topic:", msg.Topic())
	})

	client := mqtt.NewClient(opts)
	if conn := client.Connect(); conn.Wait() && conn.Error() != nil {
		panic(conn.Error())
	}

	if sub := client.Subscribe("#", 2, onMessageReceived); sub.Wait() && sub.Error() != nil {
		panic(sub.Error())
	}

	// Print the number of received messages periodically.
	go Print()

	// Wait forever.
	select {}
}
