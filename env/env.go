package env

import "os"

// Load a *required* string environment variable.
// This will panic if the variable is not set.
func load(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic("Environment variable " + name + " not set.")
	}
	return value
}

// The path under which the history files are stored, from the environment variable.
var StaticPath = load("STATIC_PATH")

// The SensorThings API base URL.
var SensorThingsBaseUrl = load("SENSORTHINGS_URL")

// The URL to the observation MQTT broker from the environment variable.
var SensorThingsObservationMqttUrl = load("MQTT_URL")

// The url to the prediction MQTT broker.
var PredictionMqttUrl = load("PREDICTION_MQTT_URL")
