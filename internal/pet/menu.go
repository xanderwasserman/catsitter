package pet

// menuEvent is one frame of input that can open or dismiss the cat menu.
type menuEvent struct {
	Right    bool
	Left     bool
	Escape   bool
	Focused  bool
	SawFocus bool
	OnMenu   bool
}

// nextMenu reports the menu after one input.
// activate is true when a menu row was chosen.
// consumed is true when the click should not also drag the cat.
func nextMenu(open bool, ev menuEvent) (openOut bool, sawFocus bool, activate bool, consumed bool) {
	if !open {
		if ev.Right {
			return true, ev.Focused, false, true
		}
		return false, false, false, false
	}
	if ev.Escape || (ev.SawFocus && !ev.Focused) {
		return false, false, false, true
	}
	if ev.Right {
		return false, false, false, true
	}
	if ev.Left {
		return false, false, ev.OnMenu, true
	}
	saw := ev.SawFocus || ev.Focused
	return true, saw, false, false
}
