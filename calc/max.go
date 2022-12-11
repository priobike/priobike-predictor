package calc

// Calculate the max of two integers.
func Max64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

// Calculate the max of two integers.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}
