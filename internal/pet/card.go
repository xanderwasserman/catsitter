package pet

import "strings"

// A card is a small panel beside the cat: About, the update offer, progress,
// a problem, or a short note. Cards are rebuilt from the view every frame.
type card struct {
	title string
	sub   string
	face  bool
	lines []cardLine
	// bar is the progress fill from 0 to 1. Negative means no bar.
	bar     float64
	buttons []cardButton
	// note cards close on any click and after a few seconds.
	note bool
}

type tone int

const (
	toneText tone = iota
	toneMuted
	toneLink
	toneWarm
)

type cardLine struct {
	// lead is plain text before text. Only text is clickable and underlined.
	lead  string
	text  string
	tone  tone
	id    string
	link  bool
	heart bool
}

func (l cardLine) width() int { return len(l.lead+l.text) * glyphW }

type cardButton struct {
	id      string
	label   string
	primary bool
}

// hit is a clickable rectangle inside a card.
type hit struct {
	box
	id string
}

// cardLayout is where everything in a card goes, relative to the card.
type cardLayout struct {
	W, H    int
	Face    box
	TitleX  int
	TitleY  int
	SubY    int
	Lines   []int
	Bar     box
	Buttons []box
	Hits    []hit
}

const (
	cardW      = 284
	cardPad    = 12
	glyphW     = 7
	lineH      = 15
	titleScale = 2
	faceSize   = 32
	buttonH    = 20
	buttonGap  = 6
	barH       = 10
	// tailW is the strip on the cat's side where the speech-bubble tail goes.
	tailW = 6
)

// cardCols is how many characters fit on one card line.
const cardCols = (cardW - tailW - 2*cardPad) / glyphW

func (c card) interactive() bool { return c.note || len(c.buttons) > 0 || c.hasLinks() }

func (c card) hasLinks() bool {
	for _, l := range c.lines {
		if l.id != "" {
			return true
		}
	}
	return false
}

// layout places the parts of c in a card body that starts at x0.
func (c card) layout(x0 int) cardLayout {
	l := cardLayout{W: cardW}
	left := x0 + cardPad
	right := x0 + cardW - tailW - cardPad
	y := cardPad
	l.TitleX = left
	if c.face {
		l.Face = box{left, y, faceSize, faceSize}
		l.TitleX = left + faceSize + 8
	}
	l.TitleY = y
	l.SubY = y + 13*titleScale + 1
	head := 13*titleScale + 1
	if c.sub != "" {
		head += lineH
	}
	if c.face && head < faceSize {
		head = faceSize
	}
	y += head + 8
	for _, line := range c.lines {
		l.Lines = append(l.Lines, y)
		if line.id != "" {
			l.Hits = append(l.Hits, hit{box{left + len(line.lead)*glyphW, y - 1, len(line.text)*glyphW + 2, lineH}, line.id})
		}
		y += lineH
	}
	if c.bar >= 0 {
		y += 4
		l.Bar = box{left, y, right - left, barH}
		y += barH + 4
	}
	if len(c.buttons) > 0 {
		y += 8
		x := left
		for _, b := range c.buttons {
			w := buttonWidth(b.label)
			if x > left && x+w > right {
				x = left
				y += buttonH + buttonGap
			}
			r := box{x, y, w, buttonH}
			l.Buttons = append(l.Buttons, r)
			l.Hits = append(l.Hits, hit{r, b.id})
			x += w + buttonGap
		}
		y += buttonH
	}
	l.H = y + cardPad
	return l
}

func buttonWidth(label string) int { return len(label)*glyphW + 16 }

// hitAt returns the id under x, y, or "".
func (l cardLayout) hitAt(x, y int) string {
	for _, h := range l.Hits {
		if h.contains(x, y) {
			return h.id
		}
	}
	return ""
}

// wrap breaks text into lines of at most cols characters, on spaces where it can.
func wrap(text string, cols int) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			for len(word) > cols {
				if line != "" {
					out = append(out, line)
					line = ""
				}
				out = append(out, word[:cols])
				word = word[cols:]
			}
			switch {
			case line == "":
				line = word
			case len(line)+1+len(word) <= cols:
				line += " " + word
			default:
				out = append(out, line)
				line = word
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// bullets wraps each item with a hanging indent and stops after max lines.
// The last kept line ends in "..." when items were cut.
func bullets(items []string, cols, max int) []string {
	var out []string
	for _, item := range items {
		for i, line := range wrap(item, cols-2) {
			prefix := "  "
			if i == 0 {
				prefix = "- "
			}
			if len(out) == max {
				last := out[max-1]
				if len(last) > cols-3 {
					last = last[:cols-3]
				}
				out[max-1] = strings.TrimRight(last, " .") + "..."
				return out
			}
			out = append(out, prefix+line)
		}
	}
	return out
}
