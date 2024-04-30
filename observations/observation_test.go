package observations

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalObservation(t *testing.T) {
	data := []byte(`
		{
			"phenomenonTime": "1970-01-01T00:00:00Z",
			"receivedTime": "1970-01-01T00:00:00Z",
			"result": 10
		}
	`)
	var o Observation
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatalf("error during unmarshal json data: %s", err.Error())
	}
	if o.PhenomenonTime.Unix() != 0 {
		t.Fatalf("error during parsing phenomenonTime: %d != 0", o.PhenomenonTime.Unix())
	}
}

func TestUnmarshalErroneousObservation(t *testing.T) {
	data := []byte(`
		{
			"phenomenonTime": "1970-01-01T00:00:00Z",
			"receivedTime": "1970-01-01T00:00:00Z",
			"result": 500
		}
	`)
	var o Observation
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatalf("error during unmarshal json data: %s", err.Error())
	}
	if o.Result != 255 {
		t.Fatalf("result should be reset to a valid value on overflow")
	}

	data = []byte(`
		{
			"phenomenonTime": "1970-01-01T00:00:00Z",
			"receivedTime": "1970-01-01T00:00:00Z",
			"result": -1
		}
	`)
	if err := json.Unmarshal(data, &o); err != nil {
		t.Fatalf("error during unmarshal json data: %s", err.Error())
	}
	if o.Result != 0 {
		t.Fatalf("result should be reset to a valid value on underflow")
	}
}
