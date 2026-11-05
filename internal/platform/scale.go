package platform

import "math"

// DIP converts a physical screen pixel into a window-position unit.
// scale is the monitor's device scale. One unit is one pixel when scale is 1.
func DIP(px int, scale float64) int {
	if scale <= 0 {
		scale = 1
	}
	return int(math.Round(float64(px) / scale))
}

// Physical is DIP in reverse: a window-position unit back to a physical pixel.
func Physical(unit int, scale float64) int {
	if scale <= 0 {
		scale = 1
	}
	return int(math.Round(float64(unit) * scale))
}

// Rect is a screen rectangle in window-position units.
type Rect struct {
	X, Y, W, H int
}

// RectDIP converts a physical rectangle, given by its edges, into window-position units.
// Converting the edges, not the size, keeps neighbouring monitors touching.
func RectDIP(left, top, right, bottom int, scale float64) Rect {
	x, y := DIP(left, scale), DIP(top, scale)
	return Rect{X: x, Y: y, W: DIP(right, scale) - x, H: DIP(bottom, scale) - y}
}
