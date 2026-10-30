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
