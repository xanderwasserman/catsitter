package pet

import (
	"fmt"
	"strings"

	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/update"
)

type cardKind int

const (
	cardNone cardKind = iota
	cardAbout
	cardOffer
	cardProgress
	cardFailed
	cardUpdated
	cardCurrent
	cardOffline
)

// Card button and link ids.
const (
	actInstall = "install"
	actLater   = "later"
	actCheck   = "check"
	actAuto    = "auto"
	actNotes   = "notes"
	actRepo    = "repo"
	actAuthor  = "author"
	actPage    = "page"
)

// noteLines is how many release-note lines the offer shows.
const noteLines = 5

// buildCard is the card for kind in this view. goos picks the macOS wording.
func buildCard(kind cardKind, view control.View, goos string) card {
	st := view.Update
	switch kind {
	case cardAbout:
		return aboutCard(st)
	case cardOffer:
		return offerCard(st, goos)
	case cardProgress:
		return progressCard(st, goos)
	case cardFailed:
		return failedCard(st)
	case cardUpdated:
		return card{
			title: "Purrfect!", sub: "Now on " + st.Version, face: true, bar: -1, note: true,
			lines: []cardLine{{text: "Updated from " + st.Updated + ".", tone: toneMuted}, {text: "Thanks for keeping me fresh."}},
		}
	case cardCurrent:
		return card{
			title: "All good", sub: st.Version + " is the newest", face: true, bar: -1, note: true,
			lines: []cardLine{{text: "Nothing to update. Back to napping."}},
		}
	case cardOffline:
		return card{
			title: "No luck", sub: "Couldn't reach GitHub", face: true, bar: -1, note: true,
			lines: []cardLine{{text: "Check your connection and try again.", tone: toneMuted}},
		}
	}
	return card{bar: -1}
}

func aboutCard(st update.Status) card {
	version := "version " + st.Version
	if !st.Enabled {
		version = "dev build (" + st.Version + ")"
	}
	c := card{title: "Catsitter", sub: version, face: true, bar: -1}
	c.lines = append(c.lines,
		cardLine{text: "A little cat that keeps your"},
		cardLine{text: "computer awake."},
		cardLine{},
		cardLine{lead: "made by ", text: control.Author, tone: toneWarm, id: actAuthor, link: true, heart: true},
		cardLine{text: strings.TrimPrefix(control.Project, "https://"), tone: toneLink, id: actRepo, link: true},
		cardLine{},
	)
	if !st.Enabled {
		c.lines = append(c.lines, cardLine{text: "Dev builds don't check for updates.", tone: toneMuted})
		c.buttons = []cardButton{{id: actLater, label: "Close", primary: true}}
		return c
	}
	mark := "[ ] "
	if st.Auto {
		mark = "[x] "
	}
	c.lines = append(c.lines, cardLine{text: mark + "Check for updates automatically", id: actAuto})
	switch label, action := control.UpdateItem(st); action {
	case control.UpdateInstall:
		c.buttons = append(c.buttons, cardButton{id: actInstall, label: label, primary: true})
	case control.UpdateOpenPage:
		c.buttons = append(c.buttons, cardButton{id: actPage, label: "Get " + st.Latest.Version.String(), primary: true})
	case control.UpdateCheck:
		c.buttons = append(c.buttons, cardButton{id: actCheck, label: "Check now"})
	}
	c.buttons = append(c.buttons, cardButton{id: actLater, label: "Close"})
	return c
}

func offerCard(st update.Status, goos string) card {
	c := card{
		title: "New version!",
		sub:   st.Version + " -> " + st.Latest.Version.String(),
		face:  true,
		bar:   -1,
	}
	notes := update.Highlights(st.Latest.Notes, noteLines)
	if len(notes) > 0 {
		c.lines = append(c.lines, cardLine{text: "What's new:", tone: toneMuted})
		for _, line := range bullets(notes, cardCols, noteLines) {
			c.lines = append(c.lines, cardLine{text: line})
		}
	} else {
		c.lines = append(c.lines, cardLine{text: "A fresh Catsitter is ready for you."})
	}
	c.lines = append(c.lines, cardLine{text: "Release notes", tone: toneLink, id: actNotes, link: true})
	if !st.CanInstall {
		c.lines = append(c.lines, cardLine{})
		for _, line := range wrap("This copy wasn't set up by the installer, so get the new one from GitHub.", cardCols) {
			c.lines = append(c.lines, cardLine{text: line, tone: toneMuted})
		}
		c.buttons = []cardButton{{id: actPage, label: "Download page", primary: true}, {id: actLater, label: "Later"}}
		return c
	}
	if goos == "darwin" {
		c.lines = append(c.lines, cardLine{})
		for _, line := range wrap("macOS will ask you to allow Accessibility again after the update.", cardCols) {
			c.lines = append(c.lines, cardLine{text: line, tone: toneWarm})
		}
	}
	c.buttons = []cardButton{{id: actInstall, label: "Update now", primary: true}, {id: actLater, label: "Later"}}
	return c
}

func progressCard(st update.Status, goos string) card {
	c := card{title: "Updating", sub: "to " + st.Latest.Version.String(), face: true, bar: st.Progress}
	if st.State == update.Downloading {
		c.lines = []cardLine{{text: fmt.Sprintf("Fetching the new me... %d%%", int(st.Progress*100))}}
		return c
	}
	c.lines = []cardLine{{text: "Checksum matches. Installing."}, {text: "Back in a blink!", tone: toneWarm}}
	if goos == "darwin" {
		c.lines = append(c.lines, cardLine{text: "Then allow Accessibility again.", tone: toneMuted})
	}
	return c
}

func failedCard(st update.Status) card {
	c := card{title: "Hmm.", sub: "The update didn't work", face: true, bar: -1}
	problem := wrap(st.Problem, cardCols)
	if len(problem) > 3 {
		problem = problem[:3]
	}
	for _, line := range problem {
		c.lines = append(c.lines, cardLine{text: line, tone: toneWarm})
	}
	c.lines = append(c.lines, cardLine{text: "Nothing changed. Still on " + st.Version + ".", tone: toneMuted})
	c.buttons = []cardButton{{id: actInstall, label: "Try again", primary: true}, {id: actPage, label: "Release page"}, {id: actLater, label: "Close"}}
	return c
}
