// Package platform describes the OS services the scheduler drives.
// Implementations live in OS-specific packages and are wired in by host.
package platform

import "time"

// Inhibitor holds a native sleep and display-sleep assertion.
// Acquire and Release are idempotent for a single holder.
type Inhibitor interface {
	Acquire() error
	Release() error
}

// IdleDetector reports how long the user has been inactive.
type IdleDetector interface {
	IdleTime() (time.Duration, error)
}

// Nudger synthesizes the tiny inputs that keep chat status online.
type Nudger interface {
	Jiggle(pixels int) error
	TapShift() error
	SwitchWindows(modifier string, count int) error
}

// Gate reports whether this process may post input.
// Trusted is always true on Linux and Windows.
type Gate interface {
	Trusted() bool
	Prompt()
}
