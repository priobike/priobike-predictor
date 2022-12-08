package observations

import (
	"encoding/json"
	"predictor/log"
	"predictor/util"
	"time"
)

// The observation model.
type Observation struct {
	// The time when the observation was made (at the site).
	PhenomenonTime time.Time `json:"phenomenonTime"`
	// The time when we received the observation.
	// Note: This isn't actually in the JSON, but we add it ourselves.
	// With this we can calculate the delay of the observation.
	ReceivedTime time.Time `json:"receivedTime"`
	// Note: The result can be a string or a number.
	// In our case, we only use observations with numbers < 255.
	// This means that we can use a byte to store the result.
	// This saves us a lot of memory and makes the code faster.
	Result byte `json:"result"`
}

// Unmarshal an observation from JSON.
func (o *Observation) UnmarshalJSON(data []byte) error {
	receivedTime := time.Now()
	var temp struct {
		PhenomenonTime time.Time `json:"phenomenonTime"`
		Result         int       `json:"result"`
	}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	o.PhenomenonTime = temp.PhenomenonTime
	o.ReceivedTime = receivedTime
	if temp.Result > 255 {
		log.Warning.Println("Observation result is too large:", temp.Result)
		temp.Result = 255
	} else if temp.Result < 0 {
		temp.Result = 0 // May happen with cycle time observations, where we don't care about the result.
	}
	o.Result = byte(temp.Result)
	return nil
}

// Flatten a list of observations into a list of seconds by their result.
func Flatten(observations []Observation) []byte {
	if len(observations) == 0 {
		return []byte{}
	}
	flattened := []byte{}
	for i := 1; i < len(observations); i++ {
		prev := observations[i-1]
		curr := observations[i]
		sDiff := int(curr.PhenomenonTime.Sub(prev.PhenomenonTime).Seconds())
		for j := 0; j < util.Min(sDiff, 300 /* Avoid too long cycles. */); j++ {
			flattened = append(flattened, prev.Result)
		}
	}
	return flattened
}
