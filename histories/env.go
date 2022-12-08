package histories

import "predictor/env"

// The path under which the history files are stored, from the environment variable.
var staticPath = env.Load("STATIC_PATH")
