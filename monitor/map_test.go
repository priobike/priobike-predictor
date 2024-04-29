package monitor

import (
	"fmt"
	"os"
	"predictor/env"
	"predictor/predictions"
	"predictor/things"
	"testing"
	"time"

	geojson "github.com/paulmach/go.geojson"
)

func TestWriteGeoJSONMap(t *testing.T) {
	laneTopology := things.LocationMultiLineString{
		Type: "MultiLineString",
		// Mock values
		Coordinates: [][][]float64{
			// Ingress lane
			{
				{
					0, 0,
				},
				{
					1, 0,
				},
			},
			// Connection lane
			{
				{
					1, 0,
				},
				{
					2, 0,
				},
			},
			// Egress lane
			{
				{
					2, 0,
				},
				{
					3, 0,
				},
			},
		},
	}
	getAllThingsForMap = func(callback func(key, value interface{}) bool) {
		callback(
			"1337_1", things.Thing{
				Name: "1337_1",
				Properties: things.ThingProperties{
					LaneType: "Radfahrer",
				},
				Locations: []things.Location{
					{
						Location: things.LocationGeoJson{
							Geometry: laneTopology,
						},
					},
				},
			},
		)
	}
	mockPrediction := predictions.Prediction{
		ThingName:     "1337_1",
		Now:           []byte{1, 1, 1, 1, 1, 3, 3, 3, 3, 3},
		NowQuality:    []byte{100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
		Then:          []byte{1, 1, 1, 1, 1, 3, 3, 3, 3, 3},
		ThenQuality:   []byte{100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
		ReferenceTime: time.Unix(0, 0),
	}
	getCurrentPredictionForMap = func(thingName string) (predictions.Prediction, bool) {
		return mockPrediction, true
	}

	tempDir := t.TempDir()
	env.StaticPath = tempDir

	locationsGeoJSONFilePath := fmt.Sprintf("%s/status/predictions-locations.geojson", tempDir)
	lanesGeoJSONFilePath := fmt.Sprintf("%s/status/predictions-lanes.geojson", tempDir)

	WriteGeoJSONMap()

	locationsData, err := os.ReadFile(locationsGeoJSONFilePath)
	if err != nil {
		t.Errorf("failed to read locations geojson file")
		t.FailNow()
	}
	lanesData, err := os.ReadFile(lanesGeoJSONFilePath)
	if err != nil {
		t.Errorf("failed to read lanes geojson file")
		t.FailNow()
	}
	locationsGeoJSON, err := geojson.UnmarshalFeatureCollection(locationsData)
	if err != nil {
		t.Errorf("failed to unmarshal geojson feature collection")
		t.FailNow()
	}
	lanesGeoJSON, err := geojson.UnmarshalFeatureCollection(lanesData)
	if err != nil {
		t.Errorf("failed to unmarshal geojson feature collection")
	}
	if len(locationsGeoJSON.Features) != 1 || len(lanesGeoJSON.Features) != 1 {
		t.Errorf("more or less than one geojson feature detected")
		t.FailNow()
	}
	unmarshaledLocationFeature := locationsGeoJSON.Features[0]
	unmarshaledLaneFeature := lanesGeoJSON.Features[0]

	type checker func(v interface{}) bool
	propertyChecks := map[string]checker{
		"prediction_available": func(v interface{}) bool {
			return v.(bool) == true
		},
		"prediction_quality": func(v interface{}) bool {
			return v.(float64) == 1.0
		},
		"prediction_time_diff": func(v interface{}) bool {
			return v.(float64) > 0
		},
		"prediction_sg_id": func(v interface{}) bool {
			return v.(string) == "1337_1"
		},
	}

	for key, check := range propertyChecks {
		v := unmarshaledLocationFeature.Properties[key]
		if !check(v) {
			t.Errorf("property check failed: %v", v)
		}
		v = unmarshaledLaneFeature.Properties[key]
		if !check(v) {
			t.Errorf("property check failed: %v", v)
		}
	}
}
