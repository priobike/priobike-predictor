package observations

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"predictor/env"
	"testing"
)

var mockResponse = `
{
	"value": [
		{
			"@iot.id": 1,
			"properties": {
				"layerName": "signal_program"
			},
			"Thing": {
				"name": "1337_1"
			},
			"Observations": [
				{
					"phenomenonTime": "2024-04-30T03:02:28Z",
					"receivedTime": "2024-04-30T03:02:28Z",
					"result": 10
				}
			]
		}
	]
}
`

func TestPrefetchMostRecentObservations(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		res.Write([]byte(mockResponse))
	}))
	defer testServer.Close()

	env.SensorThingsBaseUrlObservations = fmt.Sprintf("%s/", testServer.URL)
	PrefetchMostRecentObservations()

	signalProgramCycles.Range(func(k, v interface{}) bool {
		thingName := k.(string)
		if thingName != "1337_1" {
			t.Errorf("unexpected thing name: %s", thingName)
			t.FailNow()
		}
		cycle := v.(*Cycle)
		if cycle == nil {
			t.Errorf("no cycle parsed")
			t.FailNow()
		}
		observation, err := cycle.MakeSnapshot().GetMostRecentObservation()
		if err != nil {
			t.Errorf("last observation could not be fetched")
			t.FailNow()
		}
		if observation.Result != 10 {
			t.Errorf("unexpected program loaded")
			t.FailNow()
		}
		return true
	})
}
