package calc

import "testing"

func TestMax(t *testing.T) {
	if Max(1_000_000, 2) != 1_000_000 {
		t.Errorf("max doesn't work")
	}
	if Max(-1, 1_000_000) != 1_000_000 {
		t.Errorf("max doesn't work")
	}
}

func TestMax64(t *testing.T) {
	if Max64(1_000_000_000_000, 2) != 1_000_000_000_000 {
		t.Errorf("max64 doesn't work")
	}
	if Max64(-1, 1_000_000_000_000) != 1_000_000_000_000 {
		t.Errorf("max64 doesn't work")
	}
}
