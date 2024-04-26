package calc

import "testing"

func TestMin(t *testing.T) {
	if Min(1_000_000, 2) != 2 {
		t.Errorf("Min doesn't work")
	}
	if Min(-1, 1_000_000) != -1 {
		t.Errorf("Min doesn't work")
	}
}

func TestMin64(t *testing.T) {
	if Min64(1_000_000_000_000, 2) != 2 {
		t.Errorf("Min64 doesn't work")
	}
	if Min64(-1, 1_000_000_000_000) != -1 {
		t.Errorf("Min64 doesn't work")
	}
}
