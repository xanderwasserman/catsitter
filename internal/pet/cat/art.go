// Package cat draws the original pixel-art cat.
package cat

import (
	"image"
	"image/color"
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

func place(rows []string) *image.RGBA {
	c := newCanvas()
	c.stamp((frameW-spriteCols)/2, (frameH-len(rows))/2, rows)
	return c.image()
}

func idleFrames() []*image.RGBA {
	open := []string{
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
	half := clone(open)
	half[7] = "..KOOOOOOOOOOOOOOOOOK.."
	half[8] = "..KOOOKKKKBBKKKKKBOOOK."
	closed := clone(open)
	closed[7] = "..KOOOOOOOOOOOOOOOOOK.."
	closed[8] = "..KOOKKKKKKKKKKKKKKOOK."
	tail := clone(open)
	// The tail sits in the margin; nudge a stripe so the idle loop is visible
	// even before the blink.
	tail[5] = "...KOOOODDOOOOODDOOOK..."
	tail[15] = "....KOOODDLLLLLDDOOOK.."
	fixWidth(open, half, closed, tail)
	return []*image.RGBA{place(open), place(half), place(closed), place(tail)}
}

func sitFrames() []*image.RGBA {
	calm := []string{
		".........KKKK..........",
		".......KKPPPPKK........",
		"......KPP....PPK.......",
		".....KP..KOOOK..PK.....",
		"....KK.KOOOOOOK.KK.....",
		"...KOOOOOOOOOOOOOOOK...",
		"..KOOOOODDOOOODDOOOOK..",
		"..KOOOEEHBOOEEHBOOOOK..",
		"..KOOOEEBBOOEEBBOOOOK..",
		"..KOOOOOOOOOOOOOOOOOK..",
		"..KOOOOOWWNNWWWOOOOOK..",
		"...KOOOOOWWWWWOOOOOK...",
		"....KKOOOWWWWWOOOKK....",
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
	breathe[16] = "...K.OLLLLLLLLLLLL.OOK"
	breathe[17] = "...K.DLLLLLLLLLLLD.OOK"
	breathe[18] = "...K.OLLLLLLLLLLLL.OOK"
	fixWidth(calm, breathe)
	return []*image.RGBA{place(calm), place(breathe)}
}

func walkFrames() []*image.RGBA {
	a := []string{
		"...........KKKKK.......",
		".........KKOOOOKK......",
		"........KOODDOOOKK.....",
		".......KOODOOOOOOOK....",
		".......KOOOEEHBOOOK....",
		".......KOOOEEBBOOK.....",
		".......KOOOOONNOOK.....",
		".......KOOOWWWWOK......",
		"......KOOOWWWWOK.......",
		".....KOOLLLLLLOK.......",
		"....KOOLLLLLLLOK.......",
		"...KOODLLLLLLLOOK......",
		"..KOODDLLLLLLLOOK......",
		"..KOODLLLLLLLOOK.......",
		"...KOOOLLLLOOOK........",
		"....KOOOKKOOOK.........",
		"....KOOK..KOOK.........",
		"....KK....K.KK.........",
		"..............KK.......",
		".............KOOOK.....",
		"............KOOOOOK....",
		".............KKKKKK....",
		".......................",
		".......................",
	}
	b := clone(a)
	b[15] = "....KOKK..KKOK........."
	b[16] = "....KK.K..K.KK........."
	b[17] = "......K....K..........."
	c := clone(a)
	c[15] = ".....KOOKKOOOK........."
	c[16] = "....KK.KOOK..KK........"
	c[17] = "....K...KK............."
	d := clone(a)
	d[15] = "....KKOK..KOKK........."
	d[16] = "....K.KK..KK.K........."
	d[17] = "....K..........K......."
	d[4] = ".......KOOOEEHBOOK....."
	fixWidth(a, b, c, d)
	return []*image.RGBA{place(a), place(b), place(c), place(d)}
}

func sleepFrames() []*image.RGBA {
	base := []string{
		".........KKKK..........",
		".......KKPPPPKK........",
		"......KPP....PPK.......",
		".....KP..KOOOK..PK.....",
		"....KK.KOOOOOOK.KK.....",
		"...KOOOOOOOOOOOOOOOK...",
		"..KOOOOODDOOOODDOOOOK..",
		"..KOOKKKKKKKKKKKKKKOOK.",
		"..KOOOOOOOOOOOOOOOOOK..",
		"..KOOOOOOOOOOOOOOOOOK..",
		"..KOOOOOWWNNWWWOOOOOK..",
		"...KOOOOOWWWWWOOOOOK...",
		"....KKOOOWWWWWOOOKK....",
		"......KOOOLLLLOOOK.....",
		".....KOOLLLLLLLLOOK....",
		"....KOOLLLLLLLLLLOOOK..",
		"...KOOLLLLLLLLLLLLOOOK.",
		"...KOOLLLLLLLLLLLLLOOK.",
		"...KOOLLLLLLLLLLLLLOOK.",
		"....KOOLLLLLLLLLLOOOK..",
		".....KOOLLLLLLLLLOOK...",
		"......KOODDDDDDDOOK....",
		".......KOOO..OOOOK.....",
		"........KKK..KKKK......",
	}
	fixWidth(base)
	frames := make([]*image.RGBA, 4)
	zz := [][]string{
		{"ZZ......", ".ZZ.....", "..ZZ....", "...ZZ..."},
		{"........", "Z.......", ".Z......", "..Z....."},
	}
	for i := 0; i < 4; i++ {
		c := newCanvas()
		c.stamp((frameW-spriteCols)/2, 4, base)
		// A small Z drifting up beside the ear.
		z := []string{zz[0][i], zz[1][i]}
		c.stamp(22, 2+((3-i)%3), z)
		frames[i] = c.image()
	}
	return frames
}

func pawFrames() []*image.RGBA {
	body := []string{
		".........KKKK..........",
		".......KKPPPPKK........",
		"......KPP....PPK.......",
		".....KP..KOOOK..PK.....",
		"....KK.KOOOOOOK.KK.....",
		"...KOOOODDOOOODDOOOK...",
		"..KOOOEEHBOOEEHBOOOOK..",
		"..KOOOEEBBOOEEBBOOOOK..",
		"..KOOOOOWWNNWWWOOOOOK..",
		"...KOOOOOWWWWWOOOOOK...",
		"....KOOOOLLLLLOOOOK....",
		"...KOODOLLLLLLODOOOK...",
		"..KOOODLLLLLLLLLDOOOK..",
		"..KOODLLLLLLLLLLLDOOK..",
		".KOOODLLLLLLLLLLLLDOOK.",
	}
	keys := []string{
		".KMMMMMMMMMMMMMMMMMMMK.",
		".KMYKYMYKYMYKYMYKYMYMK.",
		".KMMMMMMMMMMMMMMMMMMMK.",
		".KKKKKKKKKKKKKKKKKKKKK.",
	}
	left := []string{
		".KOOOK..........KOOOK..",
		"..KKKK..........KKKK...",
	}
	right := []string{
		"..KOOOK......KOOOK.....",
		"...KKKK......KKKK......",
	}
	both := []string{
		".KOOOK........KOOOK....",
		"..KKKK........KKKK.....",
	}
	alt := []string{
		"...KOOOK..KOOOK........",
		"....KKKK..KKKK.........",
	}
	paws := [][]string{left, right, both, alt}
	fixWidth(body)
	fixWidth(keys)
	fixWidth(left, right, both, alt)
	frames := make([]*image.RGBA, len(paws))
	for i, paw := range paws {
		c := newCanvas()
		c.stamp((frameW-spriteCols)/2, 1, body)
		c.stamp((frameW-spriteCols)/2, 16, paw)
		c.stamp((frameW-spriteCols)/2, 20, keys)
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
func Sheet() *image.RGBA {
	const cols = 6
	order := []string{"idle", "walk", "sit", "sleep", "paws"}
	all := Frames()
	img := image.NewRGBA(image.Rect(0, 0, cols*frameW, len(order)*frameH))
	for row, name := range order {
		for col, frame := range all[name] {
			drawAt(img, frame, col*frameW, row*frameH)
		}
	}
	return img
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
