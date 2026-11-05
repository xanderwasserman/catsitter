package pet

import (
	"strings"
	"testing"
	"time"

	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/update"
)

func TestWrap(t *testing.T) {
	got := wrap("the quick brown fox jumps over the lazy dog", 10)
	want := []string{"the quick", "brown fox", "jumps over", "the lazy", "dog"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("%q", got)
	}
	if got := wrap("abcdefghijklmnop", 6); strings.Join(got, "|") != "abcdef|ghijkl|mnop" {
		t.Errorf("long word: %q", got)
	}
}

func TestBulletsIndentAndTrim(t *testing.T) {
	got := bullets([]string{"one two three four", "five"}, 12, 10)
	want := []string{"- one two", "  three four", "- five"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("%q", got)
	}
	got = bullets([]string{"aaaa", "bbbb", "cccc"}, 12, 2)
	if len(got) != 2 || !strings.HasSuffix(got[1], "...") {
		t.Errorf("trimmed: %q", got)
	}
}

func available() update.Status {
	v, _ := update.ParseVersion("0.1.3")
	return update.Status{
		State: update.Available, Version: "0.1.2", Enabled: true, CanInstall: true, Auto: true,
		Latest: update.Release{Version: v, Tag: "v0.1.3", Notes: "- A much longer line of notes that has to wrap onto the next line\n- Two\n- Three\n- Four\n- Five\n- Six", URL: "https://example/r"},
	}
}

// Every card must keep its text and buttons inside its body.
func TestCardsFit(t *testing.T) {
	statuses := map[string]update.Status{"available": available()}
	failed := available()
	failed.State = update.Failed
	failed.Problem = strings.Repeat("something went wrong ", 10)
	statuses["failed"] = failed
	dl := available()
	dl.State = update.Downloading
	dl.Progress = 0.4
	statuses["downloading"] = dl
	dev := update.Status{Version: "dev"}
	statuses["dev"] = dev
	foreign := available()
	foreign.CanInstall = false
	statuses["foreign"] = foreign
	updated := available()
	updated.Updated = "0.1.1"
	statuses["updated"] = updated

	kinds := []cardKind{cardAbout, cardOffer, cardProgress, cardFailed, cardUpdated, cardCurrent, cardOffline}
	for name, st := range statuses {
		for _, goos := range []string{"darwin", "linux", "windows"} {
			for _, kind := range kinds {
				c := buildCard(kind, control.View{Update: st}, goos)
				for _, x0 := range []int{0, tailW} {
					lay := c.layout(x0)
					right := x0 + cardW - tailW - cardPad
					for i, line := range c.lines {
						if end := x0 + cardPad + line.width(); end > right {
							t.Errorf("%s/%s/%d: line %q overflows", name, goos, kind, line.text)
						}
						if lay.Lines[i]+lineH > lay.H {
							t.Errorf("%s/%d: line %d below the card", name, kind, i)
						}
					}
					for i, b := range lay.Buttons {
						if b.X+b.W > right || b.Y+b.H > lay.H-cardPad+1 {
							t.Errorf("%s/%d: button %q outside the card", name, kind, c.buttons[i].label)
						}
					}
					if len(c.title)*glyphW*titleScale+lay.TitleX > right {
						t.Errorf("%s/%d: title %q overflows", name, kind, c.title)
					}
				}
			}
		}
	}
}

func TestCardHits(t *testing.T) {
	c := buildCard(cardOffer, control.View{Update: available()}, "linux")
	lay := c.layout(0)
	b := lay.Buttons[0]
	if id := lay.hitAt(b.X+2, b.Y+2); id != actInstall {
		t.Errorf("first button is %q", id)
	}
	if id := lay.hitAt(0, 0); id != "" {
		t.Errorf("corner hit %q", id)
	}
}

func TestOfferMentionsAccessibilityOnMac(t *testing.T) {
	text := func(goos string) string {
		var b strings.Builder
		for _, l := range buildCard(cardOffer, control.View{Update: available()}, goos).lines {
			b.WriteString(l.text + " ")
		}
		return b.String()
	}
	if !strings.Contains(text("darwin"), "Accessibility") {
		t.Error("macOS offer must warn about Accessibility")
	}
	if strings.Contains(text("windows"), "Accessibility") {
		t.Error("only macOS needs the Accessibility note")
	}
}

func TestAboutAttribution(t *testing.T) {
	c := aboutCard(available())
	var author, repo bool
	for _, l := range c.lines {
		if l.id == actAuthor && l.text == "@xanderwasserman" && l.link {
			author = true
		}
		if l.id == actRepo && l.text == "github.com/xanderwasserman/catsitter" {
			repo = true
		}
	}
	if !author || !repo {
		t.Errorf("about lines %+v", c.lines)
	}
	if c.sub != "version 0.1.2" {
		t.Errorf("sub %q", c.sub)
	}
	if dev := aboutCard(update.Status{Version: "dev"}); !strings.Contains(dev.sub, "dev build") || len(dev.buttons) != 1 {
		t.Errorf("dev about %+v", dev)
	}
}

func TestNextCard(t *testing.T) {
	now := time.Now()
	st := available()
	st.State = update.Current
	var c cardState
	c = nextCard(c, control.View{Update: st}, now)
	if c.kind != cardNone {
		t.Fatalf("first frame without an update shows %d", c.kind)
	}

	// An automatic check finding an update stays quiet.
	auto := available()
	auto.Checks = 1
	c = nextCard(c, control.View{Update: auto}, now)
	if c.kind != cardNone {
		t.Errorf("automatic check opened card %d", c.kind)
	}

	// A manual check opens the offer.
	manual := auto
	manual.Checks, manual.Manual = 2, true
	c = nextCard(c, control.View{Update: manual}, now)
	if c.kind != cardOffer || c.grab {
		t.Errorf("manual check: %+v", c)
	}
	c.close()

	busy := manual
	busy.State = update.Downloading
	c = nextCard(c, control.View{Update: busy}, now)
	if c.kind != cardProgress {
		t.Errorf("download: %d", c.kind)
	}
	fail := busy
	fail.State = update.Failed
	c = nextCard(c, control.View{Update: fail}, now)
	if c.kind != cardFailed {
		t.Errorf("failure: %d", c.kind)
	}
}

func TestNotesExpireAndRespectHiddenCat(t *testing.T) {
	now := time.Now()
	st := available()
	st.Updated = "0.1.1"
	c := nextCard(cardState{}, control.View{Update: st}, now)
	if c.kind != cardUpdated {
		t.Fatalf("after update: %d", c.kind)
	}
	c = nextCard(c, control.View{Update: st}, now.Add(noteFor+time.Second))
	if c.kind != cardNone {
		t.Errorf("note did not expire: %d", c.kind)
	}

	hidden := nextCard(cardState{}, control.View{Update: st, Hidden: true}, now)
	if hidden.kind != cardNone {
		t.Errorf("a hidden cat greeted: %d", hidden.kind)
	}
	hidden = nextCard(hidden, control.View{Update: st, Hidden: true, AboutAsks: 1}, now)
	if hidden.kind != cardAbout || !hidden.peek {
		t.Errorf("About from the tray should peek: %+v", hidden)
	}
}
