package predictions

import (
	"math"
	"predictor/calc"
	"predictor/histories"
	"predictor/observations"
	"sort"
	"time"
)

// The max cluster distance defines how far apart two cycles can be
// to be considered in the same cluster. Note that with a very
// high value, two distinct programs will be mixed together. This
// makes the prediction more robust against noise, but less agile.
// Thus we need to find a good balance. 8 Seconds is based on the
// signal group 1893_15, which will occasionally become green for
// 9 seconds.
const maxClusterDistance = 8 // Seconds

// An O(n) distance function between two phase arrays.
func distance(a []byte, b []byte) int {
	lenA := len(a)
	lenB := len(b)
	if lenA == 0 {
		return lenB
	}
	if lenB == 0 {
		return lenA
	}
	var diff = 0
	for i := 0; i < calc.Min(lenA, lenB); i++ {
		if a[i] == b[i] {
			continue
		}
		diff++
	}
	return diff + calc.Abs(lenA-lenB)
}

// Cluster a flattened history.
// Returns the clusters ordered descending by size.
func cluster(flattened [][]byte) [][][]byte {
	if len(flattened) == 0 {
		return [][][]byte{}
	}
	clusters := [][][]byte{}
	for _, colors := range flattened {
		clustered := false
		for i, cluster := range clusters {
			if distance(cluster[0], colors) < maxClusterDistance {
				clusters[i] = append(cluster, colors)
				clustered = true
				break
			}
		}
		if !clustered {
			clusters = append(clusters, [][]byte{colors})
		}
	}
	// Sort the clusters by size.
	sort.Slice(clusters, func(i, j int) bool {
		return len(clusters[i]) > len(clusters[j])
	})
	return clusters
}

// Find the best cluster with respect to a current prediction.
func best(clustered [][][]byte, current []byte) [][]byte {
	if len(clustered) == 0 {
		return [][]byte{}
	}
	if len(current) == 0 {
		return clustered[0] // Most common cluster.
	}
	// Find the best cluster.
	var bestCluster [][]byte
	var bestDistance int = math.MaxInt
	for _, cluster := range clustered {
		if len(cluster) < 1 {
			continue // We need at least one row to compare.
		}
		clusterDistance := distance(cluster[0], current)
		if clusterDistance < bestDistance {
			bestDistance = clusterDistance
			bestCluster = cluster
		}
	}
	return bestCluster
}

// Compute the most common color of a cluster by each second.
func collapse(cluster [][]byte) (values []byte, quality []byte) {
	nRows := len(cluster)
	if nRows == 0 {
		return []byte{}, []byte{}
	}

	// As the reference length, we use the shortest row.
	shortestRow := cluster[0]
	for _, row := range cluster {
		if len(row) < len(shortestRow) {
			shortestRow = row
		}
	}

	// Find the most common value in each column.
	for secondIdx := 0; secondIdx < len(shortestRow); secondIdx++ {
		var column []byte
		for _, row := range cluster {
			column = append(column, row[secondIdx])
		}
		// Make a map that contains the count of each value in the column.
		var counts = make(map[byte]float64)
		var sum float64 = 0
		for rowIdx, value := range column {
			// We add 1/(nRows-1) * row index to the value to break ties by the most recent row.
			// It is assumed that the most recent row comes last in the array.
			var tieFactor float64 = 0
			if nRows > 1 {
				tieFactor = float64(rowIdx) / float64(nRows-1)
			}
			counts[value] += 1.0 + tieFactor
			sum += 1.0 + tieFactor
		}
		// Find the biggest value in the map.
		var mostCommonValue byte
		var mostCommonCount float64
		for value, count := range counts {
			if count > mostCommonCount {
				mostCommonValue = value
				mostCommonCount = count
			}
		}
		values = append(values, mostCommonValue)
		// Find the percentage the most common value occurs in the column.
		quality = append(quality, byte(100*float64(mostCommonCount)/sum))
	}
	return values, quality
}

// Flatten observations, clamped/extended to a lower and upper time.
func flatten(observations []observations.Observation, lower time.Time, upper time.Time) []byte {
	lenObservations := len(observations)
	if lenObservations == 0 {
		return []byte{}
	}
	if lower.After(upper) {
		return []byte{}
	}
	if upper.Sub(lower) > 300*time.Second {
		upper = lower.Add(300 * time.Second) // Limit to 5 minutes.
	}
	flattened := []byte{}
	for i := 1; i < lenObservations; i++ {
		prev := observations[i-1]
		curr := observations[i]
		from := calc.Max64(lower.Unix(), prev.PhenomenonTime.Unix())
		to := calc.Min64(upper.Unix(), curr.PhenomenonTime.Unix())
		if from > to {
			continue
		}
		// If `prev` is the first observation, add the first value for
		// each second between `lower` and the `prev` observation.
		// Note that this case is not supposed to happen, as the first
		// observation should always be before `lower`.
		if i == 1 && from > lower.Unix() {
			for j := int64(0); j < from-lower.Unix(); j++ {
				flattened = append(flattened, prev.Result)
			}
		}
		// Add the previous value for each second between the previous and current observation.
		for j := int64(0); j < to-from; j++ {
			flattened = append(flattened, prev.Result)
		}
		// If `curr` is the last observation, add the last value for
		// each second between the `curr` observation and `upper``.
		if i == lenObservations-1 && to < upper.Unix() {
			for j := int64(0); j < upper.Unix()-to; j++ {
				flattened = append(flattened, curr.Result)
			}
		}
	}
	return flattened
}

// Calculate the best possible prediction for a thing.
// - Search and load the best fitting history file.
// - Load the currently running signal cycle and correlate it with clusters of the history.
// - Select the best cluster and collapse it to a prediction.
func predict(thingName string) (Prediction, error) {
	// Find the best fitting history file.
	history, programId, err := histories.LoadBestFittingHistory(thingName)
	if err != nil {
		return Prediction{}, err
	}
	// If the history is empty, we will not use it.
	if len(history.Cycles) == 0 {
		return Prediction{}, nil
	}

	// By default, we base our prediction on the last cycle end time.
	// However, the history is filtered for erroneous cycles. Therefore,
	// we use the most recent primary signal cycle end time if available.
	var runningCycleStartTime = history.Cycles[len(history.Cycles)-1].EndTime
	// Get the current primary signal cycle.
	var runningCycle = []observations.Observation{}
	primarySignalCycle, err := observations.GetPrimarySignalCycle(thingName)
	if err == nil {
		snapshot := primarySignalCycle.MakeSnapshot()

		runningCycle = snapshot.Pending
		// If the running cycle is empty, use the most recent observation.
		// This will help us when the signal is not updating its cycle but continously
		// in the same state. If we only predict based on the empty running cycle,
		// we won't have a chance of detecting this edge case.
		if len(runningCycle) == 0 {
			mostRecent, err := snapshot.GetMostRecentObservation()
			if err == nil {
				runningCycle = []observations.Observation{mostRecent}
			}
		}
		if snapshot.EndTime.After(runningCycleStartTime) { // Could be unix.Epoch if the cycle is not yet complete.
			runningCycleStartTime = snapshot.EndTime // End time of the completed cycle.
		}
	}

	// Flatten the observations into an array of signal state colors.
	// If we have observations, use them as a basis for the prediction.
	// Clamp them to the last cycle end time and now, but only if the
	// time is not too far in the past.
	var runningCycleFlat = []byte{}
	now := time.Now()
	if len(runningCycle) > 0 && now.Sub(runningCycleStartTime) < 300*time.Second {
		runningCycleFlat = flatten(runningCycle /* between */, runningCycleStartTime /* and */, now)
	}
	// Flatten the history into an array of cycles of signal state colors.
	historyFlattened := history.Flatten()
	// Cluster the history into clusters of cycles.
	clustered := cluster(historyFlattened)
	// Find the best fitting cluster, for the currently running cycle.
	bestNow := best(clustered, runningCycleFlat)
	// Collapse the best fitting clusters into a single prediction.
	predictionNow, qualitiesNow := collapse(bestNow)
	// Find the best fitting cluster, afterwards.
	bestThen := best(clustered, []byte{})
	predictionThen, qualitiesThen := collapse(bestThen)

	// Skip if no prediction could be calculated.
	if len(predictionNow) == 0 || len(predictionThen) == 0 {
		return Prediction{}, nil
	}

	// Build the prediction.
	return Prediction{
		ThingName:     thingName,
		Now:           predictionNow,
		NowQuality:    qualitiesNow,
		Then:          predictionThen,
		ThenQuality:   qualitiesThen,
		ReferenceTime: runningCycleStartTime,
		ProgramId:     programId,
	}, nil
}
