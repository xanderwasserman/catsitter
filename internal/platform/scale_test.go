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

func TestPhysicalRoundTripsAtFractionalScales(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 1.75, 2} {
		for px := -3000; px <= 3000; px++ {
			unit := DIP(px, scale)
			back := Physical(unit, scale)
			// One unit is scale pixels, so the trip lands within half a unit.
			if d := float64(back - px); d > scale/2+0.5 || d < -scale/2-0.5 {
				t.Fatalf("scale %v: %d -> %d -> %d", scale, px, unit, back)
			}
			// A second trip does not move it again, so a window cannot creep.
			if DIP(back, scale) != unit {
				t.Fatalf("scale %v: %d is not stable (%d)", scale, px, back)
			}
		}
	}
}

func TestRectDIPKeepsMonitorsTouching(t *testing.T) {
	// A 2560x1600 laptop at 150% with a 1920-wide monitor to its left.
	laptop := RectDIP(0, 0, 2560, 1600, 1.5)
	left := RectDIP(-1920, 0, 0, 1080, 1.5)
	if laptop != (Rect{0, 0, 1707, 1067}) {
		t.Fatalf("laptop %+v", laptop)
	}
	if left.X+left.W != laptop.X {
		t.Fatalf("gap between %+v and %+v", left, laptop)
	}
}
