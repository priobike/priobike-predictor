package observations

import "predictor/env"

// The SensorThings API base URL.
var baseUrl = env.Load("SENSORTHINGS_URL")

// The SensorThings query used to fetch the most recent observations.
var observationsQuery = env.Load("SENSORTHINGS_OBSERVATIONS_QUERY")

// The URL to the observation MQTT broker from the environment variable.
var observationMqttUrl = env.Load("MQTT_URL")
