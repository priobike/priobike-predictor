package predictions

import (
	"math"
	"predictor/histories"
	"predictor/observations"
	"predictor/util"
	"sort"
	"time"
)

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
	for i := 0; i < util.Min(lenA, lenB); i++ {
		if a[i] == b[i] {
			continue
		}
		diff++
	}
	return diff + util.Abs(lenA-lenB)
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
	shortestRow := cluster[0]
	for _, row := range cluster {
		if len(row) < len(shortestRow) {
			shortestRow = row
		}
	}

	// Find the most common value in each column.
	for i := 0; i < len(shortestRow); i++ {
		var column []byte
		for _, row := range cluster {
			column = append(column, row[i])
		}
		// Make a map that contains the count of each value in the column.
		var counts = make(map[byte]float64)
		var sum float64 = 0
		for _, value := range column {
			// We add 1/(nRows-1) * row index to the value to break ties by the most recent row.
			// It is assumed that the most recent row comes last in the array.
			var tieFactor float64 = 0
			if nRows > 1 {
				tieFactor = float64(i) / float64(nRows-1)
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

	// Get the current primary signal cycle.
	var runningCycle = []observations.Observation{}
	primarySignalCycle, err := observations.GetPrimarySignalCycle(thingName)
	if err == nil {
		runningCycle = primarySignalCycle.GetPending()
		// If the running cycle is empty, use the most recent observation.
		// This will help us when the signal is not updating its cycle but continously
		// in the same state. If we only predict based on the empty running cycle,
		// we won't have a chance of detecting this edge case.
		if len(runningCycle) == 0 {
			mostRecent, err := primarySignalCycle.GetMostRecentObservation()
			if err == nil {
				runningCycle = []observations.Observation{mostRecent}
			}
		}
	}

	// Flatten the observations into an array of signal state colors.
	var runningCycleFlat = []byte{}
	if len(runningCycle) > 0 && time.Since(runningCycle[len(runningCycle)-1].PhenomenonTime) < 300*time.Second {
		// If we have recent observations, use them as a basis for the prediction.
		runningCycleFlat = observations.Flatten(runningCycle)
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
	// For this we use the same technique as above, by including the last state.
	var bestThen [][]byte
	if len(predictionNow) > 0 {
		bestThen = best(clustered, []byte{predictionNow[len(predictionNow)-1]})
	} else {
		bestThen = best(clustered, []byte{})
	}
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
		ReferenceTime: history.Cycles[len(history.Cycles)-1].EndTime,
		ProgramId:     programId,
	}, nil
}
