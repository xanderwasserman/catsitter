//go:build windows

package win

import (
	"unsafe"

	"github.com/xanderwasserman/catsitter/internal/platform"
)

var procGetCursorPos = user32.NewProc("GetCursorPos")

type cursorPoint struct {
	X int32
	Y int32
}

// Pointer reports the cursor in window-position units.
// GetCursorPos is physical pixels; deviceScale converts them.
func Pointer(deviceScale float64) (int, int, bool) {
	var p cursorPoint
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0, false
	}
	return platform.DIP(int(p.X), deviceScale), platform.DIP(int(p.Y), deviceScale), true
}
