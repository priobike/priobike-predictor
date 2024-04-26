package calc

import (
	"testing"
)

func TestAbs(t *testing.T) {
	if Abs(-1) != 1 || Abs(1) != 1 {
		t.Errorf("abs doesn't work")
	}
}
