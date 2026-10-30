package x11

import "github.com/jezek/xgb/xproto"

// relativeMotion is one XTEST pointer motion.
// detail 1 asks for a relative move. detail 0 is an absolute position,
// which places the pointer at that coordinate.
func relativeMotion(dx, dy int16) (eventType byte, detail byte, root xproto.Window, x, y int16) {
	return xproto.MotionNotify, 1, 0, dx, dy
}

// stateAdd is a _NET_WM_STATE client message that adds up to two states.
// The first word is _NET_WM_STATE_ADD. A zero atom adds only the first.
// The fourth word is the application source.
func stateAdd(first, second xproto.Atom) []uint32 {
	return []uint32{1, uint32(first), uint32(second), 1, 0}
}
