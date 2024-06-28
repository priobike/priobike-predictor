package env

import (
	"fmt"
	"testing"
)

func TestLoadOptionalValidator(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()
	loadOptional("ENV_VAR", func(v string) *error {
		err := fmt.Errorf("mock error that always triggers")
		return &err
	})
}

func TestLoadRequired(t *testing.T) {
	t.Setenv("STATIC_PATH", "/usr/share/nginx/html")
	envVar := loadRequired("STATIC_PATH", func(v string) *error {
		return nil
	})
	if envVar != "/usr/share/nginx/html" {
		t.Errorf("env var not loaded correctly")
	}
}

func TestLoadRequiredPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()
	loadRequired("STATIC_PATH", func(v string) *error {
		return nil
	})
}

func TestRequiredEnvComplete(t *testing.T) {
	t.Setenv("STATIC_PATH", "/usr/share/nginx/html")
	t.Setenv("SENSORTHINGS_URL_THINGS", "https://tld.iot.hamburg.de/v1.1/")
	t.Setenv("SENSORTHINGS_URL_OBSERVATIONS", "https://tld.iot.hamburg.de/v1.1/")
	t.Setenv("SENSORTHINGS_MQTT_URL", "tcp://tld.iot.hamburg.de:1883")
	t.Setenv("PREDICTION_MQTT_URL", "tcp://predictor-mosquitto:1883")
	Init()
}

func TestRequiredEnvIncomplete(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()
	Init()
}

func TestValidatorTriggers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("should panic")
		}
	}()
	t.Setenv("STATIC_PATH", "/usr/share/nginx/html/") // TYPO
	t.Setenv("SENSORTHINGS_URL_THINGS", "https://tld.iot.hamburg.de/v1.1/")
	t.Setenv("SENSORTHINGS_URL_OBSERVATIONS", "https://tld.iot.hamburg.de/v1.1/")
	t.Setenv("SENSORTHINGS_MQTT_URL", "tcp://tld.iot.hamburg.de:1883")
	t.Setenv("PREDICTION_MQTT_URL", "tcp://predictor-mosquitto:1883")
	Init()
}

func TestValidators(t *testing.T) {
	if staticPathValidator("/test/") == nil {
		t.Errorf("static path validator should catch trailing slashes")
	}
	if sensorThingsBaseUrlValidator("https://tld.iot.hamburg.de/v1.0/") == nil {
		t.Errorf("sensorthings url validator should catch wrong api version")
	}
	if sensorThingsBaseUrlValidator("https://tld.iot.hamburg.de/v1.1") == nil {
		t.Errorf("sensorthings url validator should catch missing trailing slash")
	}
	if sensorThingsObservationMqttUrlValidator("ws://localhost:80") == nil {
		t.Errorf("sensorthings mqtt url validator should catch wrong protocol")
	}
	if predictionMqttUrlValidator("ws://localhost:80") == nil {
		t.Errorf("prediction mqtt url validator should catch wrong protocol")
	}
}
