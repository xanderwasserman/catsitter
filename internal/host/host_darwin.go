//go:build darwin

package host

import "github.com/xanderwasserman/catsitter/internal/platform/mac"

type macGate struct{}

func (macGate) Trusted() bool { return mac.Trusted() }
func (macGate) Prompt()       { mac.Prompt() }
func (macGate) Granted()      { mac.Granted() }
func (macGate) Reopen()       { mac.Reopen() }

type macRelauncher struct{}

func (macRelauncher) Relaunch() bool   { return mac.Relaunch() }
func (macRelauncher) Relaunched() bool { return mac.Relaunched() }

// Open wires CoreGraphics, IOKit, the Accessibility check, and the relaunch.
func Open() Backends {
	front := &mac.Front{}
	return Backends{
		Idle:         idleFunc(mac.IdleTime),
		Nudge:        mac.Nudger{},
		Inhibit:      mac.Inhibitor{},
		Gate:         macGate{},
		Relaunch:     macRelauncher{},
		Close:        func() {},
		Pump:         mac.Drain,
		Overlay:      mac.ExcludeFromSwitcher,
		NoteFocus:    front.Note,
		RestoreFocus: front.Restore,
		SingleThread: true,
	}
}
