package pet

// menuEvent is one frame of input that can open or dismiss the cat menu or a card.
type menuEvent struct {
	Right   bool
	Left    bool
	Escape  bool
	Outside bool
	Focused bool
	// Watch is what the open panel has seen of window focus so far.
	Watch focusWatch
	// OnMenu is true when the pointer is over the open panel.
	OnMenu bool
	// Card is true when the open panel is a card rather than the menu.
	Card bool
}

// focusWatch tracks window focus while a panel is open, so losing focus
// dismisses it only after it has had focus.
type focusWatch int

const (
	// focusUnseen: the window has not been focused since the panel opened.
	focusUnseen focusWatch = iota
	// focusSeen: the window has been focused; losing focus dismisses the panel.
	focusSeen
	// focusIgnored: the card was clicked. A link may hand focus to a browser,
	// so losing focus no longer dismisses it.
	focusIgnored
)

// watchFrom is the starting watch for a panel opened while the window has focus or not.
func watchFrom(focused bool) focusWatch {
	if focused {
		return focusSeen
	}
	return focusUnseen
}

// nextMenu reports the panel after one input.
// activate is true when a row of the menu or a spot on a card was chosen.
// The panel stays open for it; the menu row or the card action decides
// whether to close it (see menuRow.setting and chooseCard).
// A click anywhere else, Escape, a right click or losing focus closes it,
// except that a card stops closing on focus loss once it has been clicked.
// consumed is true when the click should not also drag the cat.
func nextMenu(open bool, ev menuEvent) (openOut bool, watch focusWatch, activate bool, consumed bool) {
	if !open {
		if ev.Right {
			return true, watchFrom(ev.Focused), false, true
		}
		return false, focusUnseen, false, false
	}
	blurred := ev.Watch == focusSeen && !ev.Focused
	if ev.Escape || ev.Outside || blurred {
		return false, focusUnseen, false, true
	}
	if ev.Right {
		return false, focusUnseen, false, true
	}
	if ev.Left && ev.OnMenu {
		if ev.Card {
			return true, focusIgnored, true, true
		}
		return true, ev.Watch.after(ev.Focused), true, true
	}
	if ev.Left {
		return false, focusUnseen, false, true
	}
	return true, ev.Watch.after(ev.Focused), false, false
}

// after is the watch once a frame with the given focus has passed.
func (w focusWatch) after(focused bool) focusWatch {
	if w == focusUnseen && focused {
		return focusSeen
	}
	return w
}
