//go:build windows

package win

import (
	"unsafe"

	"github.com/xanderwasserman/catsitter/internal/platform"
	"golang.org/x/sys/windows"
)

var (
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

// Focus remembers the window that had the keyboard before this one.
type Focus struct {
	ours uintptr
	prev uintptr
}

// Note records the foreground window when it is not the pet.
func (f *Focus) Note(title string) {
	hwnd := findWindow(title)
	if hwnd != 0 {
		f.ours = hwnd
	}
	fg, _, _ := procGetForegroundWindow.Call()
	if fg != 0 && fg != f.ours {
		f.prev = fg
	}
}

// Restore gives the keyboard back when the pet still holds it.
func (f *Focus) Restore() {
	fg, _, _ := procGetForegroundWindow.Call()
	target := platform.RestoreTarget(uint64(f.ours), uint64(f.prev), uint64(fg))
	if target == 0 {
		return
	}
	procSetForegroundWindow.Call(uintptr(target))
}

func findWindow(title string) uintptr {
	name, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(name)))
	return hwnd
}
