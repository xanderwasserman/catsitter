package x11

import (
	"testing"

	"github.com/jezek/xgb/xproto"
)

func TestRelativeMotionNudgesByDelta(t *testing.T) {
	typ, detail, root, x, y := relativeMotion(1, 0)
	if typ != xproto.MotionNotify || detail != 1 || root != 0 || x != 1 || y != 0 {
		t.Fatalf("out type=%d detail=%d root=%d x=%d y=%d", typ, detail, root, x, y)
	}
	_, detail, root, x, y = relativeMotion(-1, 0)
	if detail != 1 || root != 0 || x != -1 || y != 0 {
		t.Fatalf("back detail=%d root=%d x=%d y=%d", detail, root, x, y)
	}
}

func TestStateAdd(t *testing.T) {
	if got, want := stateAdd(11, 22), []uint32{1, 11, 22, 1, 0}; !sameWords(got, want) {
		t.Fatalf("pair %v", got)
	}
	if got, want := stateAdd(7, 0), []uint32{1, 7, 0, 1, 0}; !sameWords(got, want) {
		t.Fatalf("above %v", got)
	}
}

func sameWords(got, want []uint32) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
