package platform

import "testing"

func TestDIPUndoesDeviceScale(t *testing.T) {
	if got := DIP(-800, 1); got != -800 {
		t.Fatalf("scale 1: %d", got)
	}
	// 800 physical pixels on a 0.95 scale is the DIP delta the window
	// system turns back into 800 pixels.
	if got := DIP(-800, 0.95); got != -842 {
		t.Fatalf("scale 0.95: %d", got)
	}
}
