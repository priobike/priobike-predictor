package things

import "fmt"

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
	Locations   []Location   `json:"Locations"`
}

// Get the lane of a thing. This is the connection lane of the thing.
func (thing Thing) Lane() ([][]float64, error) {
	if len(thing.Locations) == 0 {
		return nil, fmt.Errorf("thing %s has no locations", thing.Name)
	}
	lanes := thing.Locations[0].Location.Geometry.Coordinates
	if len(lanes) < 2 {
		return nil, fmt.Errorf("thing %s has no ingress lane", thing.Name)
	}
	connectionLane := lanes[1] // 0: ingress lane, 1: connection lane, 2: egress lane
	if len(connectionLane) < 1 {
		return nil, fmt.Errorf("connection lane has no coordinates for thing %s", thing.Name)
	}
	return connectionLane, nil
}

// A shorthand for the traffic lights id (crossing id) of a thing.
func (t Thing) CrossingId() string {
	return t.Properties.TrafficLightsId
}

// Get the mqtt topic of a thing. This is `hamburg`/name.
func (thing Thing) Topic() string {
	return fmt.Sprintf("hamburg/%s", thing.Name)
}
