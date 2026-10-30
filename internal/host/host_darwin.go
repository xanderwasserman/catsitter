//go:build darwin

package host

import "github.com/xanderwasserman/catsitter/internal/platform/mac"

type macGate struct{}

func (macGate) Trusted() bool { return mac.Trusted() }
func (macGate) Prompt()       { mac.Prompt() }

// Open wires CoreGraphics, IOKit, and the Accessibility check.
func Open() Backends {
	front := &mac.Front{}
	return Backends{
		Idle:         idleFunc(mac.IdleTime),
		Nudge:        mac.Nudger{},
		Inhibit:      mac.Inhibitor{},
		Gate:         macGate{},
		Close:        func() {},
		Pump:         mac.Drain,
		Overlay:      mac.ExcludeFromSwitcher,
		NoteFocus:    front.Note,
		RestoreFocus: front.Restore,
	}
}
