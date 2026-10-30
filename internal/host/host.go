// Package host wires the OS backends the scheduler needs.
package host

import (
	"fmt"
	"time"

	"github.com/xanderwasserman/catsitter/internal/platform"
)

// Backends are the live platform services for this process.
type Backends struct {
	Idle    platform.IdleDetector
	Nudge   platform.Nudger
	Inhibit platform.Inhibitor
	Gate    platform.Gate
	Close   func()
	// Pump runs queued main-thread work. The pet window calls it every frame.
	// It is nil where input does not need the main thread.
	Pump func()
	// Overlay keeps the pet window out of the taskbar and the window switcher.
	// It is nil where there is nothing to change. A non-nil error means try again.
	Overlay func() error
	// MenuGrab takes keyboard and pointer input while the cat menu is open
	// and reports a press outside the window or Escape. Nil where the window
	// already receives those events.
	MenuGrab func(open bool) bool
	// Pointer returns the pointer in window-position units.
	// deviceScale is the monitor scale. ok is false when it is unavailable.
	Pointer func(deviceScale float64) (x, y int, ok bool)
	// Press reports where the pointer sat inside the window at the button press,
	// and whether the button is held. Nil where the window's own cursor is that position.
	Press func(deviceScale float64) (x, y int, pressed, down bool)
	// NoteFocus records the app that should receive keys after the menu.
	// RestoreFocus hands them back when this process still holds the keyboard.
	// Both are nil where the menu does not take focus.
	NoteFocus    func()
	RestoreFocus func()
}

type idleFunc func() (time.Duration, error)

func (f idleFunc) IdleTime() (time.Duration, error) { return f() }

// noopIdle reports the user as active so a missing idle API never nudges.
type noopIdle struct{}

func (noopIdle) IdleTime() (time.Duration, error) { return 0, nil }

type noopNudge struct{}

func (noopNudge) Jiggle(int) error { return fmt.Errorf("synthetic input is unavailable") }
func (noopNudge) TapShift() error  { return fmt.Errorf("synthetic input is unavailable") }
func (noopNudge) SwitchWindows(string, int) error {
	return fmt.Errorf("synthetic input is unavailable")
}

type openGate struct{}

func (openGate) Trusted() bool { return true }
func (openGate) Prompt()       {}
