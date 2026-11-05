package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"

	"golang.org/x/image/draw"
)

// The macOS icon grid: an 824-pixel rounded plate centred on a 1024 canvas.
const (
	canvas      = 1024
	plateInset  = 100
	plateRadius = 185
	artSize     = 704
)

var (
	plateTop    = color.RGBA{253, 243, 228, 255}
	plateBottom = color.RGBA{240, 216, 186, 255}
)

// rep is one image in the icon family, keyed by its ICNS type.
type rep struct {
	kind string
	size int
}

var reps = []rep{
	{"ic11", 32},   // 16@2x
	{"ic12", 64},   // 32@2x
	{"ic07", 128},  // 128
	{"ic13", 256},  // 128@2x
	{"ic08", 256},  // 256
	{"ic14", 512},  // 256@2x
	{"ic09", 512},  // 512
	{"ic10", 1024}, // 512@2x
}

// Compose draws the art, scaled with hard pixel edges, on the rounded plate.
func Compose(art image.Image) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, canvas, canvas))
	drawPlate(dst)
	off := (canvas - artSize) / 2
	draw.NearestNeighbor.Scale(dst, image.Rect(off, off, off+artSize, off+artSize), art, art.Bounds(), draw.Over, nil)
	return dst
}

func drawPlate(dst *image.RGBA) {
	const half = (canvas - 2*plateInset) / 2.0
	const centre = canvas / 2.0
	for y := 0; y < canvas; y++ {
		t := float64(y-plateInset) / float64(canvas-2*plateInset)
		fill := mix(plateTop, plateBottom, math.Max(0, math.Min(1, t)))
		for x := 0; x < canvas; x++ {
			d := roundedRectDistance(float64(x)+0.5-centre, float64(y)+0.5-centre, half, plateRadius)
			cover := math.Max(0, math.Min(1, 0.5-d))
			if cover == 0 {
				continue
			}
			dst.SetRGBA(x, y, scale(fill, cover))
		}
	}
}

// roundedRectDistance is the signed distance from a point, relative to the
// centre, to a square of half-width half with corners of radius r.
func roundedRectDistance(px, py, half, r float64) float64 {
	qx := math.Abs(px) - (half - r)
	qy := math.Abs(py) - (half - r)
	outside := math.Hypot(math.Max(qx, 0), math.Max(qy, 0))
	inside := math.Min(math.Max(qx, qy), 0)
	return outside + inside - r
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 255}
}

// scale premultiplies c by the coverage k.
func scale(c color.RGBA, k float64) color.RGBA {
	m := func(v uint8) uint8 { return uint8(math.Round(float64(v) * k)) }
	return color.RGBA{m(c.R), m(c.G), m(c.B), m(c.A)}
}

// Resize returns the composed icon at size pixels square.
func Resize(icon image.Image, size int) image.Image {
	if icon.Bounds().Dx() == size {
		return icon
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), icon, icon.Bounds(), draw.Src, nil)
	return dst
}

// Encode writes icon as an ICNS file with one PNG per representation.
func Encode(w io.Writer, icon image.Image) error {
	var body bytes.Buffer
	for _, r := range reps {
		var data bytes.Buffer
		if err := png.Encode(&data, Resize(icon, r.size)); err != nil {
			return err
		}
		writeChunk(&body, r.kind, data.Bytes())
	}
	var file bytes.Buffer
	writeChunk(&file, "icns", body.Bytes())
	_, err := w.Write(file.Bytes())
	return err
}

func writeChunk(w *bytes.Buffer, kind string, data []byte) {
	w.WriteString(kind)
	_ = binary.Write(w, binary.BigEndian, uint32(8+len(data)))
	w.Write(data)
}
