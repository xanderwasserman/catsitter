package pet

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"github.com/xanderwasserman/catsitter/internal/tray"
)

// The card colours come from the cat's own palette.
var (
	inkPanel  = color.RGBA{42, 33, 28, 240}
	inkEdge   = color.RGBA{246, 196, 138, 255}
	inkShade  = color.RGBA{24, 18, 16, 255}
	inkText   = color.RGBA{255, 246, 234, 255}
	inkMuted  = color.RGBA{196, 170, 146, 255}
	inkLink   = color.RGBA{126, 186, 220, 255}
	inkWarm   = color.RGBA{242, 168, 184, 255}
	inkFur    = color.RGBA{232, 138, 58, 255}
	inkFurHot = color.RGBA{246, 166, 92, 255}
	inkHeart  = color.RGBA{214, 96, 122, 255}
)

var toneInk = map[tone]color.Color{toneText: inkText, toneMuted: inkMuted, toneLink: inkLink, toneWarm: inkWarm}

var heart = []string{
	".XX.XX.",
	"XXXXXXX",
	"XXXXXXX",
	".XXXXX.",
	"..XXX..",
	"...X...",
}

// badge is the little "!" bubble the cat wears while an update waits.
var badge = []string{
	".KKKKKKK.",
	"KWWWOWWWK",
	"KWWWOWWWK",
	"KWWWOWWWK",
	"KWWWWWWWK",
	"KWWWOWWWK",
	".KKKKKKK.",
	".KWK.....",
	".K.......",
}

const badgeScale = 2

// painter draws cards with 1x1 pixel stretches and cached text images.
type painter struct {
	pixel  *ebiten.Image
	white  bool
	face   *ebiten.Image
	titles map[string]*ebiten.Image
}

func newPainter(pixel *ebiten.Image) *painter {
	return &painter{pixel: pixel, titles: map[string]*ebiten.Image{}}
}

func (p *painter) fill(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	if !p.white {
		p.pixel.Fill(color.White)
		p.white = true
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(p.pixel, op)
}

// frame is a pixel panel with notched corners and a one-pixel edge.
func (p *painter) frame(dst *ebiten.Image, b box, fill, edge color.Color) {
	p.fill(dst, b.X+2, b.Y, b.W-4, b.H, fill)
	p.fill(dst, b.X, b.Y+2, b.W, b.H-4, fill)
	p.fill(dst, b.X+1, b.Y+1, b.W-2, b.H-2, fill)
	p.fill(dst, b.X+2, b.Y, b.W-4, 1, edge)
	p.fill(dst, b.X+2, b.Y+b.H-1, b.W-4, 1, edge)
	p.fill(dst, b.X, b.Y+2, 1, b.H-4, edge)
	p.fill(dst, b.X+b.W-1, b.Y+2, 1, b.H-4, edge)
	p.fill(dst, b.X+1, b.Y+1, 1, 1, edge)
	p.fill(dst, b.X+b.W-2, b.Y+1, 1, 1, edge)
	p.fill(dst, b.X+1, b.Y+b.H-2, 1, 1, edge)
	p.fill(dst, b.X+b.W-2, b.Y+b.H-2, 1, 1, edge)
}

func (p *painter) text(dst *ebiten.Image, s string, x, y int, c color.Color) {
	text.Draw(dst, s, basicfont.Face7x13, x, y+11, c)
}

// title draws s at titleScale with square pixels.
func (p *painter) title(dst *ebiten.Image, s string, x, y int, c color.Color) {
	img, ok := p.titles[s]
	if !ok {
		img = ebiten.NewImage(len(s)*glyphW, 13)
		text.Draw(img, s, basicfont.Face7x13, 0, 11, color.White)
		p.titles[s] = img
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(titleScale, titleScale)
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(img, op)
}

func (p *painter) sprite(dst *ebiten.Image, rows []string, x, y, scale int, inks map[byte]color.Color) {
	for ry, row := range rows {
		for rx := 0; rx < len(row); rx++ {
			if c, ok := inks[row[rx]]; ok {
				p.fill(dst, x+rx*scale, y+ry*scale, scale, scale, c)
			}
		}
	}
}

func (p *painter) badge(dst *ebiten.Image, x, y int) {
	p.sprite(dst, badge, x, y, badgeScale, map[byte]color.Color{'K': inkShade, 'W': inkText, 'O': inkFur})
}

// card draws c in b. tailLeft puts the speech-bubble tail on the left edge,
// pointing at the cat. tailY is the tail's centre in b. mx, my is the pointer.
func (p *painter) card(dst *ebiten.Image, c card, b box, tailLeft bool, tailY, mx, my int) {
	x0 := 0
	body := box{b.X, b.Y, b.W - tailW, b.H}
	if tailLeft {
		x0 = tailW
		body.X += tailW
	}
	lay := c.layout(x0)
	p.frame(dst, body, inkPanel, inkEdge)
	p.tail(dst, body, tailLeft, tailY)
	at := func(r box) box { return box{b.X + r.X, b.Y + r.Y, r.W, r.H} }

	if c.face {
		if p.face == nil {
			p.face = ebiten.NewImageFromImage(tray.Face(faceSize / 16))
		}
		op := &ebiten.DrawImageOptions{}
		f := at(lay.Face)
		op.GeoM.Translate(float64(f.X), float64(f.Y))
		dst.DrawImage(p.face, op)
	}
	p.title(dst, c.title, b.X+lay.TitleX, b.Y+lay.TitleY, inkFur)
	if c.sub != "" {
		p.text(dst, c.sub, b.X+lay.TitleX, b.Y+lay.SubY, inkMuted)
	}
	left := b.X + x0 + cardPad
	for i, line := range c.lines {
		y := b.Y + lay.Lines[i]
		ink := toneInk[line.tone]
		p.text(dst, line.lead+line.text, left, y, ink)
		if line.link {
			p.fill(dst, left+len(line.lead)*glyphW, y+13, len(line.text)*glyphW, 1, ink)
		}
		if line.heart {
			p.sprite(dst, heart, left+line.width()+5, y+4, 1, map[byte]color.Color{'X': inkHeart})
		}
	}
	if c.bar >= 0 {
		p.bar(dst, at(lay.Bar), c.bar)
	}
	for i, btn := range c.buttons {
		r := at(lay.Buttons[i])
		hot := r.contains(mx, my)
		fill, edge, ink := color.Color(inkPanel), color.Color(inkEdge), color.Color(inkText)
		if btn.primary {
			fill, edge, ink = inkFur, inkShade, inkShade
			if hot {
				fill = inkFurHot
			}
		} else if hot {
			fill = color.RGBA{72, 58, 50, 255}
		}
		p.frame(dst, r, fill, edge)
		p.text(dst, btn.label, r.X+8, r.Y+4, ink)
	}
}

// bar is a segmented progress bar.
func (p *painter) bar(dst *ebiten.Image, b box, fill float64) {
	p.frame(dst, b, inkShade, inkEdge)
	inner := b.W - 4
	n := int(float64(inner) * min(max(fill, 0), 1))
	for x := 0; x < n; x += 4 {
		p.fill(dst, b.X+2+x, b.Y+2, min(3, n-x), b.H-4, inkFur)
	}
}

// tail is a stepped triangle on the cat's side of the card.
func (p *painter) tail(dst *ebiten.Image, body box, left bool, y int) {
	y = min(max(y, body.Y+10), body.Y+body.H-22)
	for i := 0; i < tailW; i++ {
		half := tailW - i
		x := body.X + body.W - 1 + i + 1
		if left {
			x = body.X - i - 1
		}
		p.fill(dst, x, y-half+1, 1, 2*half-1, inkPanel)
		p.fill(dst, x, y-half, 1, 1, inkEdge)
		p.fill(dst, x, y+half, 1, 1, inkEdge)
	}
	edgeX := body.X + body.W - 1
	if left {
		edgeX = body.X
	}
	p.fill(dst, edgeX, y-tailW+1, 1, 2*tailW-1, inkPanel)
}
