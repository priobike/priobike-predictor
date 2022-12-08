package predictions

import (
	"fmt"
	"math/rand"
	"predictor/log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// The mqtt client.
var client mqtt.Client

func ConnectPredictionPublisher() {
	log.Info.Println("Connecting to prediction mqtt broker at :", predictionMqttUrl)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(predictionMqttUrl)
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

	client = mqtt.NewClient(opts)
	if conn := client.Connect(); conn.Wait() && conn.Error() != nil {
		panic(conn.Error())
	}
}
