package predictions

import (
	"bytes"
	"time"
)

type Prediction struct {
	ThingName string `json:"thingName"`
	Now       []byte `json:"now"`  // Will be serialized to a base64 string.
	Then      []byte `json:"then"` // Will be serialized to a base64 string.
	// Certainty for the 'now' prediction part.
	NowQuality []byte `json:"nowQuality"` // Will be serialized to a base64 string.
	// Certainty for the 'then' prediction part.
	ThenQuality []byte `json:"thenQuality"` // Will be serialized to a base64 string.
	// The quality that was checked against real data.
	EvaluatedQuality int       `json:"evaluatedQuality"`
	ReferenceTime    time.Time `json:"referenceTime"`
	ProgramId        *byte     `json:"programId"`
}

// Check if a prediction equals another prediction.
func (p Prediction) Equals(other Prediction) bool {
	// First, compare things that can be compared quickly.
	if p.ThingName != other.ThingName {
		return false
	}
	if len(p.Now) != len(other.Now) {
		return false
	}
	if len(p.Then) != len(other.Then) {
		return false
	}
	if p.ReferenceTime != other.ReferenceTime {
		return false
	}
	if p.EvaluatedQuality != other.EvaluatedQuality {
		return false
	}
	if p.ProgramId == nil && other.ProgramId != nil {
		return false
	}
	if p.ProgramId != nil && other.ProgramId == nil {
		return false
	}
	if p.ProgramId != nil && other.ProgramId != nil && *p.ProgramId != *other.ProgramId {
		return false
	}
	if !bytes.Equal(p.Now, other.Now) {
		return false
	}
	if !bytes.Equal(p.Then, other.Then) {
		return false
	}
	// Don't compare the quality, to speed up the comparison.
	return true
}

// Check if a prediction is within n seconds of the other prediction.
func (p Prediction) WithinTimeOfSeconds(d time.Duration, other Prediction) bool {
	upperTimeBound := other.ReferenceTime.Add(d)
	lowerTimeBound := other.ReferenceTime.Add(-d)
	return p.ReferenceTime.After(lowerTimeBound) && p.ReferenceTime.Before(upperTimeBound)
}

// Calculate the average quality of a prediction.
func (p Prediction) AverageQuality() float64 {
	qualitySum := 0
	qualityLength := 0
	for _, quality := range p.NowQuality {
		qualitySum += int(quality)
		qualityLength++
	}
	for _, quality := range p.ThenQuality {
		qualitySum += int(quality)
		qualityLength++
	}
	if qualityLength > 0 {
		return float64(qualitySum) / float64(qualityLength)
	} else {
		return 0
	}
}
