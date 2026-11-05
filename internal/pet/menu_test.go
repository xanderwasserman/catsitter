package pet

import (
	"testing"

	"github.com/xanderwasserman/catsitter/internal/control"
)

func TestMenuClosesOnSelectionEscapeAndOutside(t *testing.T) {
	open, saw, activate, consumed := nextMenu(false, menuEvent{Right: true, Focused: true})
	if !open || !saw || activate || !consumed {
		t.Fatalf("open right: open=%v saw=%v act=%v cons=%v", open, saw, activate, consumed)
	}

	open, _, activate, consumed = nextMenu(true, menuEvent{Left: true, OnMenu: true, Focused: true, SawFocus: true})
	if open || !activate || !consumed {
		t.Fatalf("select: open=%v act=%v cons=%v", open, activate, consumed)
	}

	open, _, activate, consumed = nextMenu(true, menuEvent{Left: true, OnMenu: false, Focused: true, SawFocus: true})
	if open || activate || !consumed {
		t.Fatalf("outside: open=%v act=%v cons=%v", open, activate, consumed)
	}

	open, _, _, consumed = nextMenu(true, menuEvent{Escape: true, Focused: true, SawFocus: true})
	if open || !consumed {
		t.Fatalf("escape: open=%v cons=%v", open, consumed)
	}

	open, _, _, consumed = nextMenu(true, menuEvent{SawFocus: true, Focused: false})
	if open || !consumed {
		t.Fatalf("unfocus: open=%v cons=%v", open, consumed)
	}

	open, saw, _, consumed = nextMenu(true, menuEvent{Focused: false})
	if !open || saw || consumed {
		t.Fatalf("unfocused before focus: open=%v saw=%v cons=%v", open, saw, consumed)
	}

	open, _, _, consumed = nextMenu(true, menuEvent{Outside: true})
	if open || !consumed {
		t.Fatalf("desktop click: open=%v cons=%v", open, consumed)
	}
}

func TestMenuClosedLeftClickIsNotConsumed(t *testing.T) {
	_, _, _, consumed := nextMenu(false, menuEvent{Left: true})
	if consumed {
		t.Fatal("a click with the menu closed should still drag")
	}
}

func TestMissingAccessLeadsTheMenu(t *testing.T) {
	rows := menuRows(control.View{})
	if rows[0].id != "access" || rows[1].id != "access" {
		t.Fatalf("first rows %+v", rows[:2])
	}
	for _, row := range menuRows(control.View{Trusted: true}) {
		if row.id == "access" {
			t.Fatalf("access row while trusted: %+v", row)
		}
	}
}

func TestMenuLabelsFitTheMenu(t *testing.T) {
	const textX, advance = 12, 7
	for _, row := range menuRows(control.View{}) {
		if w := textX + len(row.label)*advance; w > menuW {
			t.Fatalf("%q is %d px wide, menu is %d", row.label, w, menuW)
		}
	}
}
