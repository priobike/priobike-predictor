package things

// A location model from the SensorThings API.
type Location struct {
	Description  string          `json:"description"`
	EncodingType string          `json:"encodingType"`
	IotId        int             `json:"@iot.id"`
	Location     LocationGeoJson `json:"location"`
}

type LocationGeoJson struct {
	Type     string                  `json:"type"`
	Geometry LocationMultiLineString `json:"geometry"`
	Name     string                  `json:"name"`
	SelfLink string                  `json:"@iot.selfLink"`
}

type LocationMultiLineString struct {
	Type        string        `json:"type"` // MultiLineString
	Coordinates [][][]float64 `json:"coordinates"`
}
