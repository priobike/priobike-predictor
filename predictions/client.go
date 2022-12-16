package predictions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"predictor/env"
	"predictor/log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// The mqtt client.
var client mqtt.Client

// The lock used for publishing to the prediction mqtt broker.
var publishLock = &sync.Mutex{}

// Publishes a prediction to the prediction MQTT broker.
func publish(p Prediction) error {
	// Acquire the lock.
	publishLock.Lock()
	defer publishLock.Unlock()

	// Publish the prediction
	topic := fmt.Sprintf("prediction/%s", p.ThingName)
	// Serialize the prediction to json.
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	dataStr := string(data)
	// Publish the prediction.
	if pub := client.Publish(topic, 2, true, dataStr); pub.Wait() && pub.Error() != nil {
		log.Error.Println("Failed to publish prediction:", pub.Error())
		return pub.Error()
	}
	return nil
}

func ConnectMQTTClient() {
	log.Info.Println("Connecting to prediction mqtt broker at :", env.PredictionMqttUrl)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(env.PredictionMqttUrl)
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
	opts.SetProtocolVersion(4)
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Warning.Println("Received unexpected message on topic:", msg.Topic())
	})

	if env.PredictionMqttUsername != "" {
		opts.SetUsername(env.PredictionMqttUsername)
	}
	if env.PredictionMqttPassword != "" {
		opts.SetPassword(env.PredictionMqttPassword)
	}

	client = mqtt.NewClient(opts)
	if conn := client.Connect(); conn.Wait() && conn.Error() != nil {
		panic(conn.Error())
	}
}
