//go:build windows

package host

import (
	"github.com/xanderwasserman/catsitter/internal/platform"
	"github.com/xanderwasserman/catsitter/internal/platform/win"
)

// Open wires Win32 idle time, SendInput, and SetThreadExecutionState.
func Open() Backends {
	focus := &win.Focus{}
	return Backends{
		Idle:         idleFunc(win.IdleTime),
		Nudge:        win.Nudger{},
		Inhibit:      &win.Inhibitor{},
		Gate:         openGate{},
		Close:        func() {},
		Overlay:      func() error { return win.ExcludeFromSwitcher(platform.OverlayTitle) },
		Pointer:      win.Pointer,
		NoteFocus:    func() { focus.Note(platform.OverlayTitle) },
		RestoreFocus: focus.Restore,
	}
}
