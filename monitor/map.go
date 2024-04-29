package monitor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"predictor/env"
	"predictor/log"
	"predictor/predictions"
	"predictor/things"
	"time"

	geojson "github.com/paulmach/go.geojson"
)

// Interface to overwrite for testing purposes.
var getAllThings = things.Things.Range

// Interface to overwrite for testing purposes.
var getCurrentPrediction = predictions.GetCurrentPrediction

// Write geojson data that can be used to visualize the predictions.
// The geojson file is written to the static directory.
func WriteGeoJSONMap() {
	// Write the geojson to the file.
	locationFeatureCollection := geojson.NewFeatureCollection() // Locations of traffic lights.
	laneFeatureCollection := geojson.NewFeatureCollection()     // Lanes of traffic lights.
	getAllThings(func(key, value interface{}) bool {
		thingName := key.(string)
		thing := value.(things.Thing)

		lane, err := thing.Lane()
		if err != nil {
			// Some things may not have lanes.
			return true
		}
		coordinate := lane[0]
		lat, lng := coordinate[1], coordinate[0]

		// Check if there is a prediction for this thing.
		prediction, predictionOk := getCurrentPrediction(thingName)
		// Build the properties.
		properties := make(map[string]interface{})
		if predictionOk {
			properties["prediction_available"] = true
			// Calculate the average quality.
			properties["prediction_quality"] = prediction.AverageQuality() / 100
			properties["prediction_time_diff"] = time.Now().Unix() - prediction.ReferenceTime.Unix()
			properties["prediction_sg_id"] = prediction.ThingName
		} else {
			properties["prediction_available"] = false
			properties["prediction_quality"] = -1
			properties["prediction_time_diff"] = 0
			properties["prediction_sg_id"] = ""
		}
		// Add thing-related properties.
		properties["thing_name"] = thing.Name
		properties["thing_properties_lanetype"] = thing.Properties.LaneType

		// Make a point feature.
		location := geojson.NewPointFeature([]float64{lng, lat})
		location.Properties = properties
		locationFeatureCollection.AddFeature(location)

		// Make a line feature.
		laneFeature := geojson.NewLineStringFeature(lane)
		laneFeature.Properties = properties
		laneFeatureCollection.AddFeature(laneFeature)

		return true
	})

	// Make sure the directory exists, otherwise create it.
	locationsFilePath := fmt.Sprintf("%s/status/predictions-locations.geojson", env.StaticPath)
	err := os.MkdirAll(filepath.Dir(locationsFilePath), os.ModePerm)
	if err != nil {
		log.Error.Println("Error creating dirs for geojson:", err)
		return
	}
	locationsGeoJson, err := locationFeatureCollection.MarshalJSON()
	if err != nil {
		log.Error.Println("Error marshalling geojson:", err)
		return
	}
	ioutil.WriteFile(locationsFilePath, locationsGeoJson, 0644)

	// Make sure the directory exists, otherwise create it.
	lanesFilePath := fmt.Sprintf("%s/status/predictions-lanes.geojson", env.StaticPath)
	err = os.MkdirAll(filepath.Dir(lanesFilePath), os.ModePerm)
	if err != nil {
		log.Error.Println("Error creating dirs for geojson:", err)
		return
	}
	lanesGeoJson, err := laneFeatureCollection.MarshalJSON()
	if err != nil {
		log.Error.Println("Error marshalling geojson:", err)
		return
	}
	ioutil.WriteFile(lanesFilePath, lanesGeoJson, 0644)
}

func UpdateGeoJSONMapPeriodically() {
	for {
		time.Sleep(30 * time.Second)
		WriteGeoJSONMap()
	}
}
