package pet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/xanderwasserman/catsitter/internal/platform"
)

// screen places the pet window. Positions and areas share one unit and origin.
type screen interface {
	position() (x, y int, ok bool)
	move(x, y int)
	// pointer is the pointer in the same units, or ok false when unknown.
	pointer() (x, y int, ok bool)
	hasPointer() bool
	// areas are the monitor areas the cat may sit on.
	areas() []box
	// desktop is true when areas cover every monitor, so a drag can be kept on screen.
	desktop() bool
	// scale is the unit's size in physical pixels, or 0 when it does not matter.
	scale() float64
	// stored converts a position to the form that is saved and kept while
	// hidden, which must not depend on the current scale. loaded reverses it.
	stored(x, y int) (int, int)
	loaded(x, y int) (int, int)
	// need is how much of the cat, in both directions, must show on one area
	// for the safety net to leave it alone.
	need() int
}

func newScreen(desk Desktop) screen {
	if desk.Place != nil {
		return placerScreen{p: desk.Place, fallback: deviceScale}
	}
	return ebitenScreen{pointAt: desk.Pointer}
}

// ebitenScreen counts from the top left of the window's current monitor.
type ebitenScreen struct {
	pointAt func(deviceScale float64) (x, y int, ok bool)
}

func (ebitenScreen) position() (int, int, bool) {
	x, y := ebiten.WindowPosition()
	return x, y, true
}

func (ebitenScreen) move(x, y int) { ebiten.SetWindowPosition(x, y) }

func (s ebitenScreen) pointer() (int, int, bool) {
	if s.pointAt == nil {
		return 0, 0, false
	}
	return s.pointAt(deviceScale())
}

func (s ebitenScreen) hasPointer() bool { return s.pointAt != nil }

func (ebitenScreen) areas() []box {
	w, h := monitorSize()
	if w <= 0 || h <= 0 {
		return nil
	}
	return []box{{0, 0, w, h}}
}

func (ebitenScreen) desktop() bool              { return false }
func (ebitenScreen) scale() float64             { return 0 }
func (ebitenScreen) stored(x, y int) (int, int) { return x, y }
func (ebitenScreen) loaded(x, y int) (int, int) { return x, y }

// need is 1 because the area is only the window's own monitor. A cat
// across a seam can show little of it, but none at all means it is lost.
func (ebitenScreen) need() int { return 1 }

// placerScreen counts from the primary monitor across the whole desktop.
// Positions are saved in physical pixels, as the unit changes size with the
// window's scale.
type placerScreen struct {
	p        *platform.Placer
	fallback func() float64
}

func (s placerScreen) position() (int, int, bool) { return s.p.Position(s.fallback()) }
func (s placerScreen) move(x, y int)              { s.p.Move(x, y, s.fallback()) }
func (s placerScreen) pointer() (int, int, bool)  { return s.p.Pointer(s.fallback()) }
func (placerScreen) hasPointer() bool             { return true }
func (placerScreen) desktop() bool                { return true }
func (s placerScreen) scale() float64             { return s.p.Scale(s.fallback()) }
func (placerScreen) need() int                    { return minVisible }

func (s placerScreen) stored(x, y int) (int, int) {
	sc := s.scale()
	return platform.Physical(x, sc), platform.Physical(y, sc)
}

func (s placerScreen) loaded(x, y int) (int, int) {
	sc := s.scale()
	return platform.DIP(x, sc), platform.DIP(y, sc)
}

func (s placerScreen) areas() []box {
	var out []box
	for _, r := range s.p.Areas(s.fallback()) {
		out = append(out, box{r.X, r.Y, r.W, r.H})
	}
	return out
}

func deviceScale() float64 {
	if mon := ebiten.Monitor(); mon != nil && mon.DeviceScaleFactor() > 0 {
		return mon.DeviceScaleFactor()
	}
	return 1
}

func monitorSize() (int, int) {
	mon := ebiten.Monitor()
	if mon == nil {
		return 0, 0
	}
	return mon.Size()
}
