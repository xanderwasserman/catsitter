// Package control is the shared surface for the tray and the cat menu.
package control

import (
	"fmt"
	"time"
)

// View is a snapshot of the settings the menus display.
type View struct {
	Paused    bool
	Inhibit   bool
	Mouse     bool
	Shift     bool
	Switch    bool
	Interval  time.Duration
	Duration  time.Duration
	Hidden    bool
	LastNudge time.Time
	WindowX   int
	WindowY   int
	WindowSet bool
	Trusted   bool
	Quitting  bool
}

// Controller is implemented by the running app.
type Controller interface {
	View() View
	TogglePause()
	SetInhibit(bool)
	SetMouse(bool)
	SetShift(bool)
	SetSwitch(bool)
	SetInterval(time.Duration)
	SetDuration(time.Duration)
	ToggleHidden()
	Quit()
	SaveWindow(x, y int)
}

// Intervals are the menu presets.
var Intervals = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
}

// Durations are the menu presets. Zero means until the user quits.
var Durations = []time.Duration{
	30 * time.Minute,
	time.Hour,
	2 * time.Hour,
	4 * time.Hour,
	8 * time.Hour,
	0,
}

// NextInterval cycles to the following preset.
func NextInterval(cur time.Duration) time.Duration {
	for i, d := range Intervals {
		if d == cur {
			return Intervals[(i+1)%len(Intervals)]
		}
	}
	return Intervals[0]
}

// NextDuration cycles to the following preset.
func NextDuration(cur time.Duration) time.Duration {
	for i, d := range Durations {
		if d == cur {
			return Durations[(i+1)%len(Durations)]
		}
	}
	return Durations[0]
}

// IntervalLabel renders a short interval name.
func IntervalLabel(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%gs", d.Seconds())
}

// DurationLabel renders a short limit name.
func DurationLabel(d time.Duration) string {
	if d <= 0 {
		return "until I stop it"
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return d.String()
}
