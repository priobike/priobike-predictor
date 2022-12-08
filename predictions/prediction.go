package predictions

import (
	"bytes"
	"time"
)

type Prediction struct {
	ThingName     string    `json:"thingName"`
	Now           []byte    `json:"now"`         // Will be serialized to a base64 string.
	NowQuality    []byte    `json:"nowQuality"`  // Will be serialized to a base64 string.
	Then          []byte    `json:"then"`        // Will be serialized to a base64 string.
	ThenQuality   []byte    `json:"thenQuality"` // Will be serialized to a base64 string.
	ReferenceTime time.Time `json:"referenceTime"`
	ProgramId     *byte     `json:"programId"`
}

// Check if a prediction equals another prediction.
func (p Prediction) Equals(other Prediction) bool {
	if p.ThingName != other.ThingName {
		return false
	}
	if len(p.Now) != len(other.Now) {
		return false
	}
	if len(p.Then) != len(other.Then) {
		return false
	}
	if !bytes.Equal(p.Now, other.Now) {
		return false
	}
	if !bytes.Equal(p.Then, other.Then) {
		return false
	}
	// Don't compare the quality, to speed up the comparison.
	if p.ReferenceTime != other.ReferenceTime {
		return false
	}
	if p.ProgramId != other.ProgramId {
		return false
	}
	return true
}

// Publishes a prediction to the prediction MQTT broker.
func (p Prediction) publish() error {
	// // Publish the prediction
	// topic := p.ThingName
	// // Serialize the prediction to json.
	// data, err := json.Marshal(p)
	// if err != nil {
	// 	return err
	// }
	// // Log the prediction.
	// log.Info.Println("Publishing prediction:", string(data))
	// // Publish the prediction.
	// if pub := client.Publish(topic, 2, true, data); pub.Wait() && pub.Error() != nil {
	// 	log.Error.Println("Failed to publish prediction:", pub.Error())
	// 	return pub.Error()
	// }
	return nil
}
