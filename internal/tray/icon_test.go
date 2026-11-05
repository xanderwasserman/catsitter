package tray

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func decode(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func alpha(img image.Image, x, y int) uint32 {
	_, _, _, a := img.At(x, y).RGBA()
	return a >> 8
}

func TestRetinaIconScalesTheArt(t *testing.T) {
	plain := decode(t, iconPNG(1, false))
	big := decode(t, iconPNG(2, false))
	if big.Bounds().Dx() != 32 || big.Bounds().Dy() != 32 {
		t.Fatalf("bounds %v", big.Bounds())
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if plain.At(x, y) != big.At(2*x+1, 2*y+1) {
				t.Fatalf("cell (%d,%d) changed at 2x", x, y)
			}
		}
	}
}

func TestRimSurroundsTheArtInsideTheCanvas(t *testing.T) {
	plain := decode(t, iconPNG(1, false))
	rimmed := decode(t, iconPNG(1, true))
	rim := 0
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if alpha(plain, x, y) == 255 && plain.At(x, y) != rimmed.At(x, y+1) {
				t.Fatalf("art cell (%d,%d) changed", x, y)
			}
			if a := alpha(rimmed, x, y); a != 0 && a != 255 {
				rim++
			}
		}
	}
	if rim == 0 {
		t.Fatal("no rim")
	}
	for x := 0; x < 16; x++ {
		if alpha(rimmed, x, 0) == 255 {
			t.Fatalf("art touches the top edge at %d", x)
		}
	}
}
