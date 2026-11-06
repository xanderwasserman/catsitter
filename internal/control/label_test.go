package control

import (
	"strings"
	"testing"
	"time"
)

func TestWatchLabel(t *testing.T) {
	if got := WatchLabel(0); got != "Watch with no time limit" {
		t.Fatalf("unlimited %q", got)
	}
	if got := WatchLabel(time.Hour); got != "Watch 1h, then pause" {
		t.Fatalf("hour %q", got)
	}
	if got := WatchLabel(30 * time.Minute); got != "Watch 30m, then pause" {
		t.Fatalf("minutes %q", got)
	}
	if got := DurationLabel(0); got != "No time limit" {
		t.Fatalf("unlimited preset %q", got)
	}
}

func TestIntervalLine(t *testing.T) {
	if got := IntervalLine(10 * time.Second); got != "Nudge after 10s idle" {
		t.Fatalf("seconds %q", got)
	}
	if got := IntervalLine(2 * time.Minute); got != "Nudge after 2m idle" {
		t.Fatalf("minutes %q", got)
	}
}

// TestTimingLinesAreOneShortRow keeps every preset's line short enough for one
// row of the cat menu, and keeps the watch line from sounding like it quits.
func TestTimingLinesAreOneShortRow(t *testing.T) {
	const most = 24
	var lines []string
	for _, d := range Intervals {
		lines = append(lines, IntervalLine(d))
	}
	for _, d := range Durations {
		lines = append(lines, WatchLabel(d))
	}
	for _, line := range lines {
		if len(line) > most {
			t.Errorf("%q is %d characters, want at most %d", line, len(line), most)
		}
		if strings.Contains(strings.ToLower(line), "quit") || strings.Contains(strings.ToLower(line), "stop") {
			t.Errorf("%q sounds like it ends Catsitter", line)
		}
	}
}
