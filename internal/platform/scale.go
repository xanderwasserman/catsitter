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
