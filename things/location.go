package things

// A location model from the SensorThings API.
type Location struct {
	Description  string   `json:"description"`
	EncodingType string   `json:"encodingType"`
	IotId        int      `json:"@iot.id"`
	Location     struct { // GeoJSON
		Type     string `json:"type"`
		Geometry struct {
			Type        string        `json:"type"` // MultiLineString
			Coordinates [][][]float64 `json:"coordinates"`
		} `json:"geometry"`
		Name     string `json:"name"`
		SelfLink string `json:"@iot.selfLink"`
	} `json:"location"`
}
