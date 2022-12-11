package monitor

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"predictor/log"
	"predictor/things"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var evaluationClient mqtt.Client

// The old prediction model.
type oldPrediction struct {
	GreentimeThreshold int64     `json:"greentimeThreshold"`
	PredictionQuality  float64   `json:"predictionQuality"`
	SignalGroupId      string    `json:"signalGroupId"`
	StartTime          time.Time `json:"startTime"`
	Value              []int64   `json:"value"`
	Timestamp          time.Time `json:"timestamp"`
}

// Parse the prediction timestamp to unix time.
func parseTimestamp(input string) (time.Time, error) {
	timestamp := strings.ReplaceAll(input, "[UTC]", "")
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// try to insert seconds
		parts := strings.Split(timestamp, "Z")
		if len(parts) != 2 {
			log.Warning.Println("Could not parse timestamp:", err)
			return time.Time{}, err
		}
		timestamp = fmt.Sprintf("%s:00Z%s", parts[0], parts[1])
		parsed, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			log.Warning.Println("Could not parse timestamp:", err)
			return time.Time{}, err
		}
	}
	return parsed, nil
}

// Unmarshal the prediction from the old service.
func (p *oldPrediction) UnmarshalJSON(data []byte) error {
	var temp struct {
		GreentimeThreshold int64   `json:"greentimeThreshold"`
		PredictionQuality  float64 `json:"predictionQuality"`
		SignalGroupId      string  `json:"signalGroupId"`
		StartTime          string  `json:"startTime"`
		Value              []int64 `json:"value"`
		Timestamp          string  `json:"timestamp"`
	}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	p.GreentimeThreshold = temp.GreentimeThreshold
	p.PredictionQuality = temp.PredictionQuality
	p.SignalGroupId = temp.SignalGroupId
	p.Value = temp.Value
	p.Timestamp, err = parseTimestamp(temp.Timestamp)
	if err != nil {
		return err
	}
	p.StartTime, err = parseTimestamp(temp.StartTime)
	if err != nil {
		return err
	}
	return nil
}

// A map that contains the current prediction for each mqtt topic.
var oldServicePredictions = &sync.Map{}

// Get the current prediction from the old service for the given thing.
func getOldServicePrediction(thingName string) (oldPrediction, bool) {
	prediction, ok := oldServicePredictions.Load(thingName)
	if !ok {
		return oldPrediction{}, false
	}
	return prediction.(oldPrediction), true
}

// An integer that represents the number of messages received.
var received uint64 = 0

// A callback that is executed when new messages arrive on the mqtt topic.
func onMessageReceived(thingName string, msg mqtt.Message) {
	atomic.AddUint64(&received, 1)
	if (received%1000) == 0 && received > 0 {
		log.Info.Printf("Received %d predictions from the old broker.", received)
	}

	// Parse the prediction from the message.
	var prediction oldPrediction
	if err := json.Unmarshal(msg.Payload(), &prediction); err != nil {
		log.Warning.Println("Could not parse prediction:", err)
		return
	}
	// Update the prediction for the connection.
	oldServicePredictions.Store(thingName, prediction)
	// Increment the number of received messages.
	atomic.AddUint64(&received, 1)
}

// Print out the number of received messages periodically.
func printReceivedPredictionsPeriodically() {
	for {
		receivedThen := atomic.LoadUint64(&received)
		time.Sleep(60 * time.Second)
		receivedNow := atomic.LoadUint64(&received)
		// If we received no messages, we re-connect to the mqtt broker.
		if receivedNow == receivedThen {
			log.Warning.Println("Reconnecting to the old prediction broker...")
			evaluationClient.Disconnect(0)
			evaluationClient.Connect()
		}
	}
}

func ListenForOldPredictions() {
	// Start a mqtt client that listens to all messages on the prediction
	// service mqtt. The mqtt broker is secured with a username and password.
	// The credentials and the mqtt url are loaded from environment variables.
	mqttUrl := os.Getenv("OLD_PREDICTION_BROKER_MQTT_URL")
	if mqttUrl == "" {
		panic("OLD_PREDICTION_BROKER_MQTT_URL not set")
	}
	log.Info.Println("Connecting to prediction mqtt broker at :", mqttUrl)

	mqttUsername := os.Getenv("OLD_PREDICTION_BROKER_MQTT_USERNAME")
	mqttPassword := os.Getenv("OLD_PREDICTION_BROKER_MQTT_PASSWORD")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)
	if mqttUsername != "" && mqttPassword != "" {
		opts.SetUsername(mqttUsername)
		opts.SetPassword(mqttPassword)
	}
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
	clientID := fmt.Sprintf("priobike-predictor-%d", random.Int())
	log.Info.Println("Using client id:", clientID)
	opts.SetClientID(clientID)
	opts.SetOrderMatters(false)
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Warning.Println("Received unexpected message on topic:", msg.Topic())
	})

	evaluationClient = mqtt.NewClient(opts)
	if conn := evaluationClient.Connect(); conn.Wait() && conn.Error() != nil {
		panic(conn.Error())
	}

	var nThings uint64
	var nSubscriptions uint64
	things.Things.Range(func(_, __ interface{}) bool {
		nThings++ // Range is performed sequentially, so no need for atomic.
		return true
	})
	var sg sync.WaitGroup
	things.Things.Range(func(thingName, _ interface{}) bool {
		sg.Add(1)
		topic := fmt.Sprintf("hamburg/%s", thingName.(string))
		go func() {
			defer sg.Done()
			atomic.AddUint64(&nSubscriptions, 1)
			if (nSubscriptions % 1000) == 0 {
				log.Info.Printf("Subscribing to %d/%d predictions on the old broker...", nSubscriptions, nThings)
			}
			if token := evaluationClient.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
				// Process the message asynchronously to avoid blocking the mqtt client.
				go onMessageReceived(thingName.(string), msg)
			}); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
		}()
		return true
	})
	sg.Wait()
	log.Info.Println("Subscribed to all predictions on the old broker.")

	go printReceivedPredictionsPeriodically()
}
