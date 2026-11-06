package pet

import (
	"time"

	"github.com/xanderwasserman/catsitter/internal/browser"
	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/update"
)

// noteFor is how long a note card stays up.
const noteFor = 7 * time.Second

// cardState is the card beside the cat, if any.
type cardState struct {
	kind cardKind
	// grab takes the pointer and keys like the menu. Only cards opened by a
	// click on the cat grab; cards that arrive on their own never do.
	grab  bool
	until time.Time
	// peek shows a hidden cat while this card is up.
	peek bool

	synced    bool
	seen      update.Status
	seenAbout int
}

func (c *cardState) show(kind cardKind, grab, peek bool) {
	c.kind, c.grab, c.peek = kind, grab, peek
	c.until = time.Time{}
	switch kind {
	case cardUpdated, cardCurrent, cardOffline:
		c.until = time.Now().Add(noteFor)
	}
}

func (c *cardState) close() { *c = cardState{synced: c.synced, seen: c.seen, seenAbout: c.seenAbout} }

// nextCard is the card to show after one view update. While the user has
// hidden the cat only About peeks out; the tray shows the rest.
func nextCard(c cardState, view control.View, now time.Time) cardState {
	st := view.Update
	if !c.synced {
		c.synced, c.seen, c.seenAbout = true, st, view.AboutAsks
		if st.Updated != "" && !view.Hidden {
			c.show(cardUpdated, false, false)
		}
		return c
	}
	prev := c.seen
	c.seen = st
	if view.AboutAsks != c.seenAbout {
		c.seenAbout = view.AboutAsks
		c.show(cardAbout, false, view.Hidden)
	}
	if st.Checks != prev.Checks && st.Manual && !view.Hidden {
		switch {
		case st.Unreachable:
			c.show(cardOffline, false, false)
		case st.State == update.Available:
			c.show(cardOffer, false, false)
		default:
			c.show(cardCurrent, false, false)
		}
	}
	if st.State == update.Failed && prev.State != update.Failed && !view.Hidden {
		c.show(cardFailed, false, false)
	}
	busy := st.State == update.Downloading || st.State == update.Installing
	switch {
	case busy && c.kind == cardNone && !view.Hidden:
		c.show(cardProgress, false, false)
	case !busy && c.kind == cardProgress:
		c.close()
	}
	if !c.until.IsZero() && now.After(c.until) {
		c.close()
	}
	return c
}

func (g *game) syncUpdates(view control.View) { g.cards = nextCard(g.cards, view, time.Now()) }

func (g *game) currentCard(view control.View) card { return buildCard(g.cards.kind, view, g.goos) }

// panelShown is true when the menu or a card is beside the cat.
func (g *game) panelShown() bool { return g.menu || g.cards.kind != cardNone }

// inputOpen is true when clicks and Escape go to the menu or a card.
func (g *game) inputOpen(view control.View) bool {
	return g.menu || (g.cards.kind != cardNone && g.currentCard(view).interactive())
}

func (g *game) grabbing() bool { return g.menu || (g.cards.kind != cardNone && g.cards.grab) }

func (g *game) badgeShown(view control.View) bool {
	return !g.panelShown() && view.Update.State == update.Available
}

func (g *game) panelSize(view control.View) (int, int) {
	if g.menu {
		return menuW, menuPad*2 + len(menuRows(view))*rowH
	}
	if g.cards.kind != cardNone {
		lay := g.currentCard(view).layout(0)
		return lay.W, lay.H
	}
	return 0, 0
}

// chooseCard handles a click at x, y inside the card.
func (g *game) chooseCard(x, y int, view control.View) {
	c := g.currentCard(view)
	x0 := 0
	if g.lay.Menu.X > g.lay.Cat.X {
		x0 = tailW
	}
	id := c.layout(x0).hitAt(x, y)
	st := view.Update
	switch id {
	case actInstall:
		g.cards.close()
		g.ctrl.InstallUpdate()
	case actCheck:
		g.cards.close()
		g.ctrl.CheckForUpdates()
	case actAuto:
		g.ctrl.SetAutoUpdate(!st.Auto)
	case actNotes:
		browser.Open(st.Latest.URL)
	case actRepo:
		browser.Open(control.Project)
	case actAuthor:
		browser.Open(control.AuthorURL)
	case actPage:
		g.cards.close()
		browser.Open(st.Latest.URL)
	case actLater:
		g.cards.close()
	case "":
		if c.note {
			g.cards.close()
		}
	}
}

func (g *game) runUpdateItem(view control.View) {
	if _, action := control.UpdateItem(view.Update); action != g.drawnUpdate {
		return
	}
	switch g.drawnUpdate {
	case control.UpdateCheck:
		g.ctrl.CheckForUpdates()
	case control.UpdateInstall:
		g.ctrl.InstallUpdate()
	case control.UpdateOpenPage:
		g.cards.show(cardOffer, true, false)
	}
}
