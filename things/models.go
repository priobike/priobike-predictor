package things

import (
	"fmt"
)

// A traffic light thing from the SensorThings API.
type Thing struct {
	IotId       int    `json:"@iot.id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Properties  struct {
		LaneType        string `json:"laneType"`
		TrafficLightsId string `json:"trafficLightsId"` // This is the crossing.
	} `json:"properties"`
	Datastreams []Datastream `json:"Datastreams"`
}

// A shorthand for the traffic lights id (crossing id) of a thing.
func (t Thing) CrossingId() string {
	return t.Properties.TrafficLightsId
}

// A traffic light datastream model from the SensorThings API.
type Datastream struct {
	IotId       int    `json:"@iot.id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Properties  struct {
		LayerName string `json:"layerName"`
	} `json:"properties"`
}

// Get the MQTT topic for a datastream.
func (d Datastream) MqttTopic() string {
	return fmt.Sprintf("v1.1/Datastreams(%d)/Observations", d.IotId)
}
