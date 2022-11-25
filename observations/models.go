package observations

import (
	"encoding/json"
	"time"
)

// The observation model.
type Observation struct {
	PhenomenonTime time.Time `json:"phenomenonTime"`
	// Note: The result can be a string or a number.
	// In our case, we only use observations with numbers.
	Result int `json:"result"`
}

// Unmarshal the observation from the message.
// The phenomenonTime and resultTime come in the format: "2022-11-24T04:02:03.000Z"
func (o *Observation) UnmarshalJSON(data []byte) error {
	type Alias Observation
	aux := &struct {
		PhenomenonTime string `json:"phenomenonTime"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	parsedPhenomenonTime, err := time.Parse(time.RFC3339, aux.PhenomenonTime)
	if err != nil {
		return err
	}
	o.PhenomenonTime = parsedPhenomenonTime
	return nil
}
