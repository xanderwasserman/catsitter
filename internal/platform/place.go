package platform

// Placer moves the pet window in desktop-wide window-position units, in place
// of Ebitengine's SetWindowPosition, which counts from the top left of the
// window's current monitor. Every hook takes Ebitengine's device scale, used
// only when the system cannot report the window's own.
type Placer struct {
	// Scale is the window's device scale, physical pixels per unit.
	Scale func(deviceScale float64) float64
	// Position is the window's top-left corner.
	Position func(deviceScale float64) (x, y int, ok bool)
	// Move puts the window's top-left corner at x, y without resizing it.
	Move func(x, y int, deviceScale float64)
	// Pointer is the pointer anywhere on the desktop.
	Pointer func(deviceScale float64) (x, y int, ok bool)
	// Areas are the work areas of every monitor.
	Areas func(deviceScale float64) []Rect
}
