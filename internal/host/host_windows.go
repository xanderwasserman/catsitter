//go:build windows

package host

import "github.com/xanderwasserman/catsitter/internal/platform/win"

// Open wires Win32 idle time, SendInput, and SetThreadExecutionState.
func Open() Backends {
	return Backends{
		Idle:    idleFunc(win.IdleTime),
		Nudge:   win.Nudger{},
		Inhibit: &win.Inhibitor{},
		Gate:    openGate{},
		Close:   func() {},
	}
}
