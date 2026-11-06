package pet

import (
	"testing"

	"github.com/xanderwasserman/catsitter/internal/control"
)

func TestMenuStaysOpenOnSelectionAndClosesOnEscapeAndOutside(t *testing.T) {
	open, saw, activate, consumed := nextMenu(false, menuEvent{Right: true, Focused: true})
	if !open || !saw || activate || !consumed {
		t.Fatalf("open right: open=%v saw=%v act=%v cons=%v", open, saw, activate, consumed)
	}

	open, saw, activate, consumed = nextMenu(true, menuEvent{Left: true, OnMenu: true, Focused: true, SawFocus: true})
	if !open || !saw || !activate || !consumed {
		t.Fatalf("select: open=%v saw=%v act=%v cons=%v", open, saw, activate, consumed)
	}

	open, _, activate, consumed = nextMenu(true, menuEvent{Left: true, OnMenu: false, Focused: true, SawFocus: true})
	if open || activate || !consumed {
		t.Fatalf("outside: open=%v act=%v cons=%v", open, activate, consumed)
	}

	open, _, activate, consumed = nextMenu(true, menuEvent{Escape: true, Focused: true, SawFocus: true})
	if open || activate || !consumed {
		t.Fatalf("escape: open=%v act=%v cons=%v", open, activate, consumed)
	}

	open, _, activate, consumed = nextMenu(true, menuEvent{Right: true, Focused: true, SawFocus: true})
	if open || activate || !consumed {
		t.Fatalf("right click: open=%v act=%v cons=%v", open, activate, consumed)
	}

	open, _, _, consumed = nextMenu(true, menuEvent{SawFocus: true, Focused: false})
	if open || !consumed {
		t.Fatalf("unfocus: open=%v cons=%v", open, consumed)
	}

	open, saw, _, consumed = nextMenu(true, menuEvent{Focused: false})
	if !open || saw || consumed {
		t.Fatalf("unfocused before focus: open=%v saw=%v cons=%v", open, saw, consumed)
	}

	open, _, activate, consumed = nextMenu(true, menuEvent{Outside: true, Left: true, OnMenu: true})
	if open || activate || !consumed {
		t.Fatalf("desktop click: open=%v act=%v cons=%v", open, activate, consumed)
	}
}

// TestRepeatedChoicesKeepTheMenuOpen clicks the menu several times in a row,
// as when cycling the idle time, then clicks away.
func TestRepeatedChoicesKeepTheMenuOpen(t *testing.T) {
	open, saw, _, _ := nextMenu(false, menuEvent{Right: true, Focused: true})
	for i := 0; i < 5; i++ {
		var activate bool
		open, saw, activate, _ = nextMenu(open, menuEvent{Left: true, OnMenu: true, Focused: true, SawFocus: saw})
		if !open || !activate {
			t.Fatalf("click %d: open=%v act=%v", i, open, activate)
		}
		open, saw, _, _ = nextMenu(open, menuEvent{Focused: true, SawFocus: saw})
		if !open {
			t.Fatalf("closed after click %d", i)
		}
	}
	if open, _, _, _ = nextMenu(open, menuEvent{Left: true, Focused: true, SawFocus: saw}); open {
		t.Fatal("click away left the menu open")
	}
}

// TestSettingsKeepTheMenuOpen checks which rows leave the menu up when chosen:
// the settings do, and rows that leave the menu do not.
func TestSettingsKeepTheMenuOpen(t *testing.T) {
	settings := map[string]bool{
		"pause": true, "inhibit": true, "mouse": true, "shift": true,
		"switch": true, "interval": true, "duration": true,
	}
	view := control.View{Update: available()}
	seen := map[string]bool{}
	for _, row := range menuRows(view) {
		seen[row.id] = true
		if row.setting != settings[row.id] {
			t.Errorf("row %q setting=%v, want %v", row.id, row.setting, settings[row.id])
		}
	}
	for id := range settings {
		if !seen[id] {
			t.Errorf("no %q row", id)
		}
	}
	for _, id := range []string{"access", "hide", "update", "about", "quit"} {
		if !seen[id] {
			t.Errorf("no %q row to check", id)
		}
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
	if rows[0].id != "paused" || rows[1].id != "access" {
		t.Fatalf("first rows %+v", rows[:2])
	}
	for _, row := range menuRows(control.View{Trusted: true}) {
		if row.id == "paused" || row.id == "access" {
			t.Fatalf("access row while trusted: %+v", row)
		}
	}
}

func TestMenuLabelsFitTheMenu(t *testing.T) {
	const textX, advance = 12, 7
	for _, iv := range control.Intervals {
		for _, d := range control.Durations {
			for _, row := range menuRows(control.View{Interval: iv, Duration: d}) {
				if w := textX + len(row.label)*advance; w > menuW {
					t.Fatalf("%q is %d px wide, menu is %d", row.label, w, menuW)
				}
			}
		}
	}
}

func TestMenuHasUpdateAndAbout(t *testing.T) {
	ids := func(v control.View) []string {
		var out []string
		for _, r := range menuRows(v) {
			out = append(out, r.id)
		}
		return out
	}
	dev := ids(control.View{Trusted: true})
	if dev[len(dev)-2] != "about" || dev[len(dev)-1] != "quit" {
		t.Errorf("dev rows %v", dev)
	}
	for _, id := range dev {
		if id == "update" {
			t.Error("dev builds show no update row")
		}
	}
	rows := menuRows(control.View{Trusted: true, Update: available()})
	row := rows[len(rows)-3]
	if row.id != "update" || row.label != "Update to 0.1.3" || !row.hot {
		t.Errorf("update row %+v", row)
	}
	for _, r := range rows {
		if len(r.label)*7+12 > menuW {
			t.Errorf("row %q is wider than the menu", r.label)
		}
	}
}
