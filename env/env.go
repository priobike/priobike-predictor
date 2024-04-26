package env

import (
	"fmt"
	"os"
	"strings"
)

// Load a *required* string environment variable.
// This will panic if the variable is not set.
func loadRequired(name string, validate func(string) *error) string {
	value := os.Getenv(name)
	if value == "" {
		panic("Environment variable " + name + " not set.")
	}
	if err := validate(value); err != nil {
		panic(err)
	}
	return value
}

// Load am *optional* string environment variable.
// This will return an empty string if the variable is not set.
func loadOptional(name string, validate func(string) *error) string {
	value := os.Getenv(name)
	if err := validate(value); err != nil {
		panic(err)
	}
	return value
}

// The path under which the history files are stored, from the environment variable.
var StaticPath string

// The SensorThings API base URL.
var SensorThingsBaseUrl string

// The URL to the observation MQTT broker from the environment variable.
var SensorThingsObservationMqttUrl string

// The url to the prediction MQTT broker.
var PredictionMqttUrl string

// The username to use for the prediction MQTT broker.
var PredictionMqttUsername string

// The password to use for the prediction MQTT broker.
var PredictionMqttPassword string

var staticPathValidator = func(value string) *error {
	if strings.HasSuffix(value, "/") {
		err := fmt.Errorf("static path shouldn't end with a slash")
		return &err
	}
	return nil
}

var sensorThingsBaseUrlValidator = func(value string) *error {
	if !strings.HasSuffix(value, "/v1.1/") {
		err := fmt.Errorf("unknown sensorthings api version or missing trailing slash")
		return &err
	}
	return nil
}

var sensorThingsObservationMqttUrlValidator = func(value string) *error {
	if !strings.HasPrefix(value, "tcp://") {
		err := fmt.Errorf("only tcp protocol for sensorthings mqtt broker supported right now")
		return &err
	}
	return nil
}

var predictionMqttUrlValidator = func(value string) *error {
	if !strings.HasPrefix(value, "tcp://") {
		err := fmt.Errorf("only tcp protocol for prediction mqtt broker supported right now")
		return &err
	}
	return nil
}

var emptyValidator = func(value string) *error {
	return nil
}

func Init() {
	StaticPath = loadRequired("STATIC_PATH", staticPathValidator)
	SensorThingsBaseUrl = loadRequired("SENSORTHINGS_URL", sensorThingsBaseUrlValidator)
	SensorThingsObservationMqttUrl = loadRequired("SENSORTHINGS_MQTT_URL", sensorThingsObservationMqttUrlValidator)
	PredictionMqttUrl = loadRequired("PREDICTION_MQTT_URL", predictionMqttUrlValidator)
	PredictionMqttUsername = loadOptional("PREDICTION_MQTT_USERNAME", emptyValidator)
	PredictionMqttPassword = loadOptional("PREDICTION_MQTT_PASSWORD", emptyValidator)
}
