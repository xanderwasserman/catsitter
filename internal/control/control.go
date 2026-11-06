// Package control is the shared surface for the tray and the cat menu.
package control

import (
	"fmt"
	"time"

	"github.com/xanderwasserman/catsitter/internal/update"
)

// Author, AuthorURL and Project are shown in About.
const (
	Author    = "@xanderwasserman"
	AuthorURL = "https://github.com/xanderwasserman"
	Project   = "https://github.com/xanderwasserman/catsitter"
)

// View is a snapshot of the settings the menus display.
type View struct {
	Paused   bool
	Inhibit  bool
	Mouse    bool
	Shift    bool
	Switch   bool
	Interval time.Duration
	Duration time.Duration
	Hidden   bool
	// Recall counts the requests to bring the cat back on screen.
	Recall    uint64
	LastNudge time.Time
	WindowX   int
	WindowY   int
	WindowSet bool
	Trusted   bool
	Quitting  bool
	Update    update.Status
	// AboutAsks counts requests to show About, so the cat can tell a new one.
	AboutAsks int
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
	// BringBack shows the cat and puts it at its default spot on screen.
	BringBack()
	// RequestAccess explains the missing input permission and offers to fix it.
	RequestAccess()
	CheckForUpdates()
	InstallUpdate()
	SetAutoUpdate(bool)
	// ShowAbout asks the cat to show its About panel.
	ShowAbout()
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

// Durations are the menu presets. Zero means no time limit.
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

// IntervalLine is the menu line for how long the user must be idle before a nudge.
func IntervalLine(d time.Duration) string {
	return "Nudge after " + IntervalLabel(d) + " idle"
}

// DurationLabel renders a short limit name.
func DurationLabel(d time.Duration) string {
	if d <= 0 {
		return "No time limit"
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return d.String()
}

// WatchLabel is the menu line for how long Catsitter keeps watch before it
// pauses by itself.
func WatchLabel(d time.Duration) string {
	if d <= 0 {
		return "Watch with no time limit"
	}
	return "Watch " + DurationLabel(d) + ", then pause"
}

// UpdateAction is what the update menu item does when chosen.
type UpdateAction int

const (
	UpdateNone UpdateAction = iota
	UpdateCheck
	UpdateInstall
	// UpdateOpenPage opens the release page, for copies the installer does not manage.
	UpdateOpenPage
)

// UpdateItem is the update line both menus show. An empty label means no line.
func UpdateItem(st update.Status) (string, UpdateAction) {
	if !st.Enabled {
		return "", UpdateNone
	}
	switch st.State {
	case update.Downloading:
		return fmt.Sprintf("Updating... %d%%", int(st.Progress*100)), UpdateNone
	case update.Installing:
		return "Installing " + st.Latest.Version.String() + "...", UpdateNone
	case update.Available, update.Failed:
		if !st.CanInstall {
			return "Get Catsitter " + st.Latest.Version.String() + "...", UpdateOpenPage
		}
		return "Update to " + st.Latest.Version.String(), UpdateInstall
	}
	if st.Checking {
		return "Checking for updates...", UpdateNone
	}
	return "Check for updates", UpdateCheck
}

// VersionLine is the attribution line, for example "Catsitter 0.1.2 by @xanderwasserman".
func VersionLine(version string) string {
	if !update.IsRelease(version) {
		version = "dev build"
	}
	return "Catsitter " + version + " by " + Author
}
