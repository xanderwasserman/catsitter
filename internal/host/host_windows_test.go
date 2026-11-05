package host

import "testing"

// The overlay and focus hooks change the pet window from Update. Win32
// delivers those changes to the thread that owns the window, which only
// reads its messages when Update runs there.
func TestWindowsRunsUpdateOnTheWindowThread(t *testing.T) {
	if !Open().SingleThread {
		t.Fatal("SingleThread is off")
	}
}
