package cat

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestFramesAreCats(t *testing.T) {
	all := Frames()
	for _, name := range []string{"idle", "walk", "sit", "sleep", "paws"} {
		frames := all[name]
		if len(frames) < 2 {
			t.Fatalf("%s has %d frames", name, len(frames))
		}
		for i, f := range frames {
			if f.Bounds().Dx() != frameW || f.Bounds().Dy() != frameH {
				t.Fatalf("%s frame %d size %v", name, i, f.Bounds())
			}
			if _, _, _, a := f.At(0, 0).RGBA(); a != 0 {
				t.Fatalf("%s frame %d corner is opaque", name, i)
			}
			if !hasOpaque(f) {
				t.Fatalf("%s frame %d is blank", name, i)
			}
		}
	}
	if sameImage(all["idle"][0], all["paws"][0]) {
		t.Fatal("paw frame matches idle")
	}
	if sameImage(all["walk"][0], all["walk"][1]) {
		t.Fatal("walk frames do not move")
	}
	if sameImage(all["sleep"][0], all["sleep"][2]) {
		t.Fatal("sleep frames do not change")
	}
}

// TestBlinkIsSlowAndRare plays idle at the app's six frames a second: the
// eyes stay open for most of a loop of at least four seconds, then close
// once and stay shut for at least half a second.
func TestBlinkIsSlowAndRare(t *testing.T) {
	const fps = 6
	idle := Frames()["idle"]
	if len(idle) < 4*fps {
		t.Fatalf("idle loop is %d frames, want at least %d", len(idle), 4*fps)
	}
	wide := eyePixels(idle[0])
	if wide == 0 {
		t.Fatal("idle starts with its eyes shut")
	}
	var open, blinks, shut, longestShut int
	for i, f := range idle {
		eyes := eyePixels(f)
		if eyes == wide {
			open++
		} else if eyePixels(idle[(i+len(idle)-1)%len(idle)]) == wide {
			blinks++
		}
		if eyes == 0 {
			shut++
			longestShut = max(longestShut, shut)
		} else {
			shut = 0
		}
	}
	if open*4 < len(idle)*3 {
		t.Fatalf("eyes open for %d of %d frames, want at least three quarters", open, len(idle))
	}
	if blinks != 1 {
		t.Fatalf("idle blinks %d times a loop, want once", blinks)
	}
	if longestShut < fps/2 {
		t.Fatalf("eyes shut for %d frames, want at least %d", longestShut, fps/2)
	}
}

// eyePixels counts the iris and pupil pixels in a frame.
func eyePixels(img *image.RGBA) int {
	n := 0
	for i := 0; i < len(img.Pix); i += 4 {
		c := color.RGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
		if c == palette[pEye] || c == palette[pPupil] {
			n++
		}
	}
	return n
}

func TestSheetMatchesFile(t *testing.T) {
	got := Sheet()
	path := filepath.Join("sheet.png")
	if os.Getenv("UPDATE_SPRITES") == "1" {
		writePNG(t, path, got)
		writePNG(t, filepath.Join("..", "..", "..", "packaging", "icon.png"), Icon(64))
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("sprite sheet missing (run with UPDATE_SPRITES=1): %v", err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if !samePixels(got, img) {
		t.Fatal("sheet.png does not match the pixel data")
	}
}

func hasOpaque(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				return true
			}
		}
	}
	return false
}

func sameImage(a, b image.Image) bool { return samePixels(a, b) }

func samePixels(a, b image.Image) bool {
	if a.Bounds().Dx() != b.Bounds().Dx() || a.Bounds().Dy() != b.Bounds().Dy() {
		return false
	}
	bnd := a.Bounds()
	bb := b.Bounds()
	for y := 0; y < bnd.Dy(); y++ {
		for x := 0; x < bnd.Dx(); x++ {
			ar, ag, ab, aa := a.At(bnd.Min.X+x, bnd.Min.Y+y).RGBA()
			br, bg, bbv, ba := b.At(bb.Min.X+x, bb.Min.Y+y).RGBA()
			if ar != br || ag != bg || ab != bbv || aa != ba {
				return false
			}
		}
	}
	return true
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPosesShareTheGround(t *testing.T) {
	all := Frames()
	ground := lowestRow(all["idle"][0])
	for name, frames := range all {
		for i, f := range frames {
			if got := lowestRow(f); got != ground {
				t.Fatalf("%s frame %d stands on row %d, idle on %d", name, i, got, ground)
			}
		}
	}
}

func lowestRow(img image.Image) int {
	b := img.Bounds()
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				return y
			}
		}
	}
	return -1
}
