package things

import "predictor/env"

// The SensorThings API base URL.
var baseUrl = env.Load("SENSORTHINGS_URL")

// The SensorThings query used to fetch the Things.
var thingsQuery = env.Load("SENSORTHINGS_THINGS_QUERY")
