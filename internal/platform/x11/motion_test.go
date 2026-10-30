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

func TestSkipStateDataAddsBothStates(t *testing.T) {
	got := skipStateData(11, 22)
	want := []uint32{1, 11, 22, 1, 0}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v", got)
		}
	}
}
