package env

import "os"

// Load a *required* string environment variable.
// This will panic if the variable is not set.
func Load(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic("Environment variable " + name + " not set.")
	}
	return value
}
