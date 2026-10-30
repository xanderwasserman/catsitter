package platform

import "testing"

func TestRestoreTarget(t *testing.T) {
	if got := RestoreTarget(2, 9, 2); got != 9 {
		t.Fatalf("held focus should return to %d, got %d", 9, got)
	}
	if got := RestoreTarget(2, 9, 9); got != 0 {
		t.Fatalf("focus already moved: %d", got)
	}
	if got := RestoreTarget(2, 0, 2); got != 0 {
		t.Fatalf("nothing saved: %d", got)
	}
	if got := RestoreTarget(0, 9, 0); got != 0 {
		t.Fatalf("no window: %d", got)
	}
}
