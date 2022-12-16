package env

import "os"

// Load a *required* string environment variable.
// This will panic if the variable is not set.
func loadRequired(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic("Environment variable " + name + " not set.")
	}
	return value
}

// Load am *optional* string environment variable.
// This will return an empty string if the variable is not set.
func loadOptional(name string) string {
	return os.Getenv(name)
}

// The path under which the history files are stored, from the environment variable.
var StaticPath = loadRequired("STATIC_PATH")

// The SensorThings API base URL.
var SensorThingsBaseUrl = loadRequired("SENSORTHINGS_URL")

// The URL to the observation MQTT broker from the environment variable.
var SensorThingsObservationMqttUrl = loadRequired("SENSORTHINGS_MQTT_URL")

// The url to the prediction MQTT broker.
var PredictionMqttUrl = loadRequired("PREDICTION_MQTT_URL")

// The username to use for the prediction MQTT broker.
var PredictionMqttUsername = loadOptional("PREDICTION_MQTT_USERNAME")

// The password to use for the prediction MQTT broker.
var PredictionMqttPassword = loadOptional("PREDICTION_MQTT_PASSWORD")
