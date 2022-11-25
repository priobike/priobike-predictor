package things

// A things response from the SensorThings API.
type ThingsResponse struct {
	Value   []Thing `json:"value"`
	NextUri *string `json:"@iot.nextLink"`
}
