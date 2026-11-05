//go:build windows

package host

import (
	"github.com/xanderwasserman/catsitter/internal/platform"
	"github.com/xanderwasserman/catsitter/internal/platform/win"
)

// Open wires Win32 idle time, SendInput, and SetThreadExecutionState.
func Open() Backends {
	focus := &win.Focus{}
	window := win.NewWindow(platform.OverlayTitle)
	return Backends{
		Idle:         idleFunc(win.IdleTime),
		Nudge:        win.Nudger{},
		Inhibit:      &win.Inhibitor{},
		Gate:         openGate{},
		Close:        func() {},
		Overlay:      func() error { return win.ExcludeFromSwitcher(platform.OverlayTitle) },
		Place:        window.Placer(),
		NoteFocus:    func() { focus.Note(platform.OverlayTitle) },
		RestoreFocus: focus.Restore,
		SingleThread: true,
	}
}
