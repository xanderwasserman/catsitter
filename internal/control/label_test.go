package control

import (
	"strings"
	"testing"
	"time"
)

func TestWatchLabel(t *testing.T) {
	if got := WatchLabel(0); got != "Until I stop it" || strings.Contains(got, "For ") {
		t.Fatalf("unlimited %q", got)
	}
	if got := WatchLabel(time.Hour); got != "For 1h" {
		t.Fatalf("hour %q", got)
	}
	if got := WatchLabel(30 * time.Minute); got != "For 30m" {
		t.Fatalf("minutes %q", got)
	}
}
