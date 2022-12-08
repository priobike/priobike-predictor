package predictions

import "predictor/env"

// The path under which the history files are stored, from the environment variable.
var staticPath = env.Load("STATIC_PATH")

// The url to the prediction MQTT broker.
var predictionMqttUrl = env.Load("PREDICTION_MQTT_URL")
