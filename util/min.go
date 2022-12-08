package util

// Calculate the min of two integers.
func Min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// Calculate the min of two integers.
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
