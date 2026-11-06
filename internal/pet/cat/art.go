// Package cat draws the original pixel-art cat.
package cat

import (
	"image"
	"image/color"
	"strings"
)

const (
	frameW = 32
	frameH = 32
	Scale  = 3
)

const (
	pNone byte = iota
	pLine
	pFur
	pDark
	pCream
	pWhite
	pPink
	pNose
	pEye
	pPupil
	pShine
	pZ
	pBoard
	pKey
)

var palette = []color.RGBA{
	pNone:  {0, 0, 0, 0},
	pLine:  {42, 32, 28, 255},
	pFur:   {232, 138, 58, 255},
	pDark:  {176, 88, 36, 255},
	pCream: {246, 196, 138, 255},
	pWhite: {255, 246, 234, 255},
	pPink:  {242, 168, 184, 255},
	pNose:  {214, 96, 122, 255},
	pEye:   {54, 120, 72, 255},
	pPupil: {28, 22, 26, 255},
	pShine: {236, 244, 186, 255},
	pZ:     {126, 186, 220, 255},
	pBoard: {62, 48, 40, 255},
	pKey:   {228, 214, 196, 255},
}

type canvas struct {
	pix []byte
}

func newCanvas() *canvas {
	return &canvas{pix: make([]byte, frameW*frameH)}
}

func (c *canvas) set(x, y int, v byte) {
	if x < 0 || y < 0 || x >= frameW || y >= frameH || v == pNone {
		return
	}
	c.pix[y*frameW+x] = v
}

func (c *canvas) image() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, frameW, frameH))
	for i, v := range c.pix {
		img.SetRGBA(i%frameW, i/frameW, palette[v])
	}
	return img
}

// stamp paints a row string onto the canvas. Unknown runes are skipped.
// Legend: . transparent, K outline, O fur, D stripe, L cream, W white,
// P pink, N nose, E eye, B pupil, H shine, Z sleep, M board, Y key.
func (c *canvas) stamp(x, y int, rows []string) {
	for dy, row := range rows {
		for dx, r := range row {
			c.set(x+dx, y+dy, runeColor(r))
		}
	}
}

func runeColor(r rune) byte {
	switch r {
	case 'K':
		return pLine
	case 'O':
		return pFur
	case 'D':
		return pDark
	case 'L':
		return pCream
	case 'W':
		return pWhite
	case 'P':
		return pPink
	case 'N':
		return pNose
	case 'E':
		return pEye
	case 'B':
		return pPupil
	case 'H':
		return pShine
	case 'Z':
		return pZ
	case 'M':
		return pBoard
	case 'Y':
		return pKey
	default:
		return pNone
	}
}

// Hand-drawn sheets. Each row is 24 columns and is centered in the 32px frame.
const spriteCols = 24

// frontHead is the sitting cat's face without its eyes; frontCat stamps them on.
var frontHead = []string{
	".........KKKK..........",
	".......KKPPPPKK........",
	"......KPP....PPK.......",
	".....KP..KOOOK..PK.....",
	"....KK.KOOOOOOK.KK.....",
	"...KOOOODDOOOODDOOOK...",
	"..KOOOOOOOOOOOOOOOOOK..",
	"..KOOOOOOOOOOOOOOOOOK..",
	"..KOOOOOOOOOOOOOOOOOK..",
	"..KOOOOOOOOOOOOOOOOOK..",
	"..KOOOOOWWNNWWWOOOOOK..",
	"...KOOOOOWWWWWOOOOOK...",
	"....KKOOOWWWWWOOOKK....",
}

// Eyes are stamped over the face's fur, so . leaves the fur showing.
var (
	eyesOpen = []string{
		".EEEE.",
		"EWWBBE",
		"EWBBHE",
		".EEEE.",
	}
	eyesHalf = []string{
		"......",
		".KKKK.",
		"KWBBHK",
		".EEEE.",
	}
	eyesClosed = []string{
		"......",
		"......",
		"K....K",
		".KKKK.",
	}
)

// frontCat draws the sitting cat with its face above body and the given eyes.
func frontCat(body, eyes []string) *image.RGBA {
	const eyeY, leftEyeX, rightEyeX = 6, 4, 13
	rows := append(clone(frontHead), body...)
	fixWidth(rows)
	c := newCanvas()
	x, y := (frameW-spriteCols)/2, (frameH-len(rows))/2
	c.stamp(x, y, rows)
	c.stamp(x+leftEyeX, y+eyeY, eyes)
	c.stamp(x+rightEyeX, y+eyeY, eyes)
	return c.image()
}

// idleFrames is the sitting cat looking out with big round eyes. It holds
// still, twitches, then gives one slow blink, at six frames a second.
func idleFrames() []*image.RGBA {
	body := []string{
		"......KOOOLLLLOOOK.....",
		".....KOODDLLLLDDOOK....",
		"....KOOODLLLLLLLDOOOK..",
		"...KOOOODLLLLLLLDOOOOK.",
		"...KOODDDLLLLLLLDDDOOK.",
		"...KOOODLLLLLLLLLDOOOK.",
		"....KOOODLLLLLLLDOOOK..",
		".....KOOODLLLLLDOOOK...",
		"......KOODDDDDDDOOK....",
		".......KOOO..OOOOK.....",
		"........KKK..KKKK......",
	}
	twitch := clone(body)
	twitch[2] = "....KOOODDLLLLLDDOOOK.."
	open := frontCat(body, eyesOpen)
	half := frontCat(body, eyesHalf)
	closed := frontCat(body, eyesClosed)
	return sequence(
		hold(open, 9),
		hold(frontCat(twitch, eyesOpen), 2),
		hold(open, 9),
		hold(half, 1),
		hold(closed, 3),
		hold(half, 2),
	)
}

func sitFrames() []*image.RGBA {
	calm := []string{
		"......KOOOLLLLOOOK.....",
		".....KOOLLLLLLLLOOK....",
		"....KOOLLLLLLLLLLOOOK..",
		"...KOOLLLLLLLLLLLLOOOK.",
		"...KODLLLLLLLLLLLLDOOK.",
		"...KOOLLLLLLLLLLLLLOOK.",
		"....KOOLLLLLLLLLLOOOK..",
		".....KOOLLLLLLLLLOOK...",
		"......KOODDDDDDDOOK....",
		".......KOOO..OOOOK.....",
		"........KKK..KKKK......",
	}
	breathe := clone(calm)
	breathe[3] = "...K.OLLLLLLLLLLLL.OOK"
	breathe[4] = "...K.DLLLLLLLLLLLD.OOK"
	breathe[5] = "...K.OLLLLLLLLLLLL.OOK"
	return []*image.RGBA{frontCat(calm, eyesOpen), frontCat(breathe, eyesOpen)}
}

// hold repeats a frame so it stays on screen for n frames.
func hold(frame *image.RGBA, n int) []*image.RGBA {
	out := make([]*image.RGBA, n)
	for i := range out {
		out[i] = frame
	}
	return out
}

func sequence(parts ...[]*image.RGBA) []*image.RGBA {
	var out []*image.RGBA
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// walkFrames is a side view of the cat walking right: a stride, a passing
// step, the other stride and the other passing step. The far legs are
// stripe-dark and sit a little ahead of the near ones, and the body rises
// a pixel on each passing step. The feet stay on the other poses' ground.
func walkFrames() []*image.RGBA {
	const rows, farShift = 24, 2
	head := []string{
		".............KK....KK...",
		"............KPPK..KPPK..",
		"............KPPOKKOPPK..",
		"...........KOOOOOOOOOOK.",
		"...........KOODDOOOOOOK.",
		"...........KOOOOOOEHBOK.",
		"...........KOOOOOOEBBOOK",
		"...........KOOOOOOOOOWNK",
		"...........KOOOOOOOWWWK.",
		"............KOOOOWWWWK..",
		"..............OLLLLWK...",
		"..................KK....",
	}
	body := []string{
		"....KKKKKKKKKK..........",
		"...KOODOOODOOOKK........",
		"..KOOODOOODOOOOOK.......",
		"..KOOOOOOOOOOOOOOK......",
		"..KOOOOOOOOOOOOOLK......",
		"..KOOOLLLLLLLLLLLK......",
		"...KOLLLLLLLLLLLK.......",
		"....KKKKKKKKKKKK........",
	}
	tails := [][]string{
		{".KK.", "KOOK", "KODK", "KOOK", ".KOK", ".KDOK", "..KOOK", "...KOK"},
		{"..KK", ".KOOK", ".KODK", ".KOOK", ".KOK", ".KDOK", "..KOOK", "...KOK"},
	}
	spread := []string{
		"...KOOK.......KOOK......",
		"..KOOK.........KOOK.....",
		"..KOOK.........KOOK.....",
		".KOOK...........KOOK....",
		".KOOK...........KOOK....",
		"KOOOK...........KOOOK...",
		"KKKK.............KKKK...",
	}
	under := []string{
		"......KOOK...KOOK.......",
		"......KOOK...KOOK.......",
		"......KOOK...KOOK.......",
		"......KOOK...KOOK.......",
		"......KOOK...KOOK.......",
		"......KOOOK..KOOOK......",
		"......KKKK...KKKK.......",
	}
	hindUp := []string{
		"....KOOK.....KOOK.......",
		"....KOOK.....KOOK.......",
		"...KOOK......KOOK.......",
		"...KOOOK.....KOOK.......",
		"....KKKK.....KOOK.......",
		".............KOOOK......",
		".............KKKK.......",
	}
	foreUp := []string{
		".....KOOK....KOOK.......",
		".....KOOK....KOOK.......",
		".....KOOK.....KOOK......",
		".....KOOK.....KOOOK.....",
		".....KOOK......KKKK.....",
		".....KOOOK..............",
		".....KKKK...............",
	}
	type pose struct {
		near, far  []string
		rise, tail int
	}
	poses := []pose{
		{spread, darken(under), 0, 0},
		{hindUp, darken(under), 1, 1},
		{under, darken(spread), 0, 0},
		{under, darken(foreUp), 1, 1},
	}
	frames := make([]*image.RGBA, len(poses))
	for i, p := range poses {
		c := newCanvas()
		x, y := (frameW-spriteCols)/2, (frameH-rows)/2-p.rise
		c.stamp(x+farShift, y+17, stretch(p.far, p.rise))
		c.stamp(x, y+3, tails[p.tail])
		c.stamp(x, y+10, body)
		c.stamp(x, y+17, stretch(p.near, p.rise))
		c.stamp(x, y+3, head)
		frames[i] = c.image()
	}
	return frames
}

// stretch lengthens legs at the hip so they still meet a raised body.
func stretch(rows []string, by int) []string {
	out := make([]string, 0, len(rows)+by)
	for i := 0; i < by; i++ {
		out = append(out, rows[0])
	}
	return append(out, rows...)
}

// darken redraws fur as stripe so a leg reads as the far one.
func darken(rows []string) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = strings.ReplaceAll(row, "O", "D")
	}
	return out
}

// sleepFrames is the cat curled up on its side with its tail wrapped round
// the front and its head on its paws. Its back rises and falls, and a Z
// drifts up from its head.
func sleepFrames() []*image.RGBA {
	body := []string{
		"......KKKKKKKK..........",
		"....KKOODOODOOKK........",
		"...KOODOOODOOODOK.......",
		"..KOODOOODOOODOOOK......",
		"..KOOOOOOOOOOOOOOOK.....",
		".KOOOOOOOOOOOOOOOOK.....",
		".KOOOOOOOOOOOOOOOOOK....",
		".KOOOOOOOOOOOOOOOOOK....",
		".KOOOOOOOOOOOOOOOOOK....",
		"..KOOOOOOOOOOOOOOOOK....",
		"...KKKKKKKKKKKKKKKK.....",
	}
	tail := []string{
		"..KKKKKKKKKKKKK.........",
		".KOODOODOODOOOOK........",
		"..KKKKKKKKKKKKK.........",
	}
	head := []string{
		".............KK....KK...",
		"............KPPK..KPPK..",
		"............KPPOKKOPPK..",
		"...........KOOOOOOOOOOK.",
		"...........KOODDOOOOOOK.",
		"...........KOOOOOKOOKOK.",
		"...........KOOOOOOKKOOOK",
		"...........KOOOOOOOOOWNK",
		"...........KOOOOOOOOWWWK",
		"..........KLLLKOOOWWWWK.",
		"..........KKKKKKKKKKKK..",
	}
	small := []string{"ZZZZ", "..Z.", ".Z..", "ZZZZ"}
	big := []string{"ZZZZZ", "...Z.", "..Z..", ".Z...", "ZZZZZ"}
	// The back rises by repeating a row through the middle of the body.
	breath := append(clone(body[:6]), body[5:]...)
	type step struct {
		in     bool
		z      []string
		zx, zy int
	}
	steps := []step{
		{false, small, 24, 9},
		{true, small, 25, 7},
		{true, big, 25, 4},
		{true, big, 26, 2},
		{false, big, 26, 0},
		{false, nil, 0, 0},
	}
	const ground = 27
	frames := make([]*image.RGBA, len(steps))
	for i, st := range steps {
		back := body
		if st.in {
			back = breath
		}
		c := newCanvas()
		x := (frameW - spriteCols) / 2
		c.stamp(x, ground+1-len(back), back)
		c.stamp(x, ground+1-len(tail), tail)
		c.stamp(x, ground+1-len(head), head)
		c.stamp(st.zx, st.zy, st.z)
		frames[i] = c.image()
	}
	return frames
}

// pawFrames is the sitting cat behind a keyboard, tapping it with one paw
// while the other is raised. Pale marks flick off the keys under each tap.
func pawFrames() []*image.RGBA {
	body := []string{
		".........KKKK..........",
		".......KKPPPPKK........",
		"......KPP....PPK.......",
		".....KP..KOOOK..PK.....",
		"....KK.KOOOOOOK.KK.....",
		"...KOOOODDOOOODDOOOK...",
		"..KOOOOOOOOOOOOOOOOOK..",
		"..KOOOEEHBOOEEHBOOOOK..",
		"..KOOOEEBBOOEEBBOOOOK..",
		"..KOOOOOOOOOOOOOOOOOK..",
		"..KOOOOOWWNNWWWOOOOOK..",
		"...KOOOOOWWWWWOOOOOK...",
		"....KKOOOWWWWWOOOKK....",
		"...KOODOLLLLLLODOOOK...",
		"..KOOODLLLLLLLLLDOOOK..",
	}
	keyboard := []string{
		"....KKKKKKKKKKKKKKKKKKKKKKKK....",
		"...KMMMMMMMMMMMMMMMMMMMMMMMMK...",
		"...KMYYMYYMYYMYYMYYMYYMYYMMMK...",
		"..KMMMMMMMMMMMMMMMMMMMMMMMMMMK..",
		"..KMMYYMYYMYYMYYMYYMYYMYYMYYMK..",
		".KMMMMMMMMMMMMMMMMMMMMMMMMMMMMK.",
		".KMYYMYYYYYYYYYYYYYYYYYYYYMYYMK.",
		".KMMMMMMMMMMMMMMMMMMMMMMMMMMMMK.",
		".KKKKKKKKKKKKKKKKKKKKKKKKKKKKKK.",
	}
	down := []string{
		"KOOOOK",
		"KOOOOK",
		"KOOOOK",
		"KOOOOK",
		"KLLLLK",
		".KKKK.",
	}
	up := []string{
		".KKKK.",
		"KOOOOK",
		"KOOOOK",
		"KLLLLK",
		"KKOOKK",
		".KOOK.",
		".KOOK.",
	}
	tap := []string{
		"H......H",
		".H....H.",
	}
	// A raised paw lifts clear of the keys and swings out past the cheek.
	type side struct{ downX, upX int }
	left, right := side{6, 4}, side{20, 22}
	const downY, upY = 18, 12
	type hit struct{ leftUp, rightUp bool }
	hits := []hit{{false, true}, {false, false}, {true, false}, {false, false}}
	frames := make([]*image.RGBA, len(hits))
	for i, h := range hits {
		c := newCanvas()
		c.stamp((frameW-spriteCols)/2, 4, body)
		c.stamp(0, 19, keyboard)
		for _, p := range []struct {
			side
			up, taps bool
		}{{left, h.leftUp, h.rightUp}, {right, h.rightUp, h.leftUp}} {
			if p.up {
				c.stamp(p.upX, upY, up)
				continue
			}
			c.stamp(p.downX, downY, down)
			if p.taps {
				c.stamp(p.downX-1, downY+2, tap)
			}
		}
		frames[i] = c.image()
	}
	return frames
}

func clone(rows []string) []string {
	out := make([]string, len(rows))
	copy(out, rows)
	return out
}

// fixWidth pads or crops every row to spriteCols so a missed column
// cannot shift the cat sideways.
func fixWidth(groups ...[]string) {
	for _, rows := range groups {
		for i, row := range rows {
			for len(row) < spriteCols {
				row += "."
			}
			if len(row) > spriteCols {
				row = row[:spriteCols]
			}
			rows[i] = row
		}
	}
}

// Frames returns every animation, keyed by name.
func Frames() map[string][]*image.RGBA {
	return map[string][]*image.RGBA{
		"idle":  idleFrames(),
		"walk":  walkFrames(),
		"sit":   sitFrames(),
		"sleep": sleepFrames(),
		"paws":  pawFrames(),
	}
}

// Sheet packs the animations into one sprite sheet, six frames per row.
// A held frame is drawn once.
func Sheet() *image.RGBA {
	const cols = 6
	order := []string{"idle", "walk", "sit", "sleep", "paws"}
	all := Frames()
	img := image.NewRGBA(image.Rect(0, 0, cols*frameW, len(order)*frameH))
	for row, name := range order {
		for col, frame := range poses(all[name]) {
			drawAt(img, frame, col*frameW, row*frameH)
		}
	}
	return img
}

// poses drops repeats of the frame before, leaving each drawing once in order.
func poses(frames []*image.RGBA) []*image.RGBA {
	var out []*image.RGBA
	for i, f := range frames {
		if i == 0 || f != frames[i-1] {
			out = append(out, f)
		}
	}
	return out
}

func drawAt(dst *image.RGBA, src *image.RGBA, x, y int) {
	for py := 0; py < src.Bounds().Dy(); py++ {
		for px := 0; px < src.Bounds().Dx(); px++ {
			dst.Set(x+px, y+py, src.At(px, py))
		}
	}
}

// Icon is a nearest-neighbor scaled front view for the tray and desktop.
func Icon(size int) *image.RGBA {
	src := idleFrames()[0]
	out := image.NewRGBA(image.Rect(0, 0, size, size))
	b := src.Bounds()
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			out.Set(x, y, src.At(x*b.Dx()/size, y*b.Dy()/size))
		}
	}
	return out
}
