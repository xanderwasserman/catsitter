package pet

import "testing"

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
}

func TestMenuClosedLeftClickIsNotConsumed(t *testing.T) {
	_, _, _, consumed := nextMenu(false, menuEvent{Left: true})
	if consumed {
		t.Fatal("a click with the menu closed should still drag")
	}
}
