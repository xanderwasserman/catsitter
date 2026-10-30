//go:build linux

package host

import (
	"log"

	"github.com/xanderwasserman/catsitter/internal/platform"
	"github.com/xanderwasserman/catsitter/internal/platform/inhibit"
	"github.com/xanderwasserman/catsitter/internal/platform/x11"
)

// Open connects to X11 and prepares a sleep inhibitor.
// A missing X server (typical of a pure Wayland session) leaves nudges off.
func Open() Backends {
	b := Backends{
		Idle:    noopIdle{},
		Nudge:   noopNudge{},
		Inhibit: &inhibit.Inhibitor{},
		Gate:    openGate{},
		Close:   func() {},
	}
	client, err := x11.Open()
	if err != nil {
		log.Printf("X11 is unavailable (%v). Mouse and key nudges stay off. Wayland can still use the sleep inhibitor; synthetic input needs X11 or XWayland.", err)
		return b
	}
	b.Idle = client
	b.Nudge = client
	b.Close = client.Close
	b.Overlay = func() error { return client.ExcludeFromSwitcher(platform.OverlayTitle) }
	b.MenuGrab = func(open bool) bool { return client.MenuGrab(platform.OverlayTitle, open) }
	b.Pointer = client.Pointer
	b.Press = func(scale float64) (int, int, bool, bool) {
		return client.Press(platform.OverlayTitle, scale)
	}
	return b
}
