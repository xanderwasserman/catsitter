//go:build darwin

package host

import (
	"path/filepath"

	"github.com/xanderwasserman/catsitter/internal/platform/mac"
)

type macGate struct{}

func (macGate) Trusted() bool { return mac.Trusted() }
func (macGate) Prompt()       { mac.Prompt() }
func (macGate) Granted()      { mac.Granted() }
func (macGate) Reopen()       { mac.Reopen() }

// Open wires CoreGraphics, IOKit, the Accessibility check, and the relaunch.
// dir is Catsitter's config directory.
func Open(dir string) Backends {
	front := &mac.Front{}
	return Backends{
		Idle:         idleFunc(mac.IdleTime),
		Nudge:        mac.Nudger{},
		Inhibit:      mac.Inhibitor{},
		Gate:         macGate{},
		Relaunch:     mac.Relauncher{Marker: filepath.Join(dir, "relaunch")},
		Close:        func() {},
		Pump:         mac.Drain,
		Overlay:      mac.ExcludeFromSwitcher,
		NoteFocus:    front.Note,
		RestoreFocus: front.Restore,
		SingleThread: true,
	}
}
