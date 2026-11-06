package cat

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xanderwasserman/catsitter/internal/pet"
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
	if sleep := poses(all["sleep"]); sameImage(sleep[0], sleep[2]) {
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

// TestSleepIsUnhurried plays sleep at the app's six frames a second: one
// breath and one drifting Z take at least four seconds, and every pose stays
// up for more than half a second, so the loop reads as slow breathing.
func TestSleepIsUnhurried(t *testing.T) {
	const fps = 6
	sleep := Frames()["sleep"]
	if len(sleep) < 4*fps {
		t.Fatalf("sleep loop is %d frames (%.1fs), want at least %d", len(sleep), float64(len(sleep))/fps, 4*fps)
	}
	if n := len(poses(sleep)); n != 6 {
		t.Fatalf("sleep has %d poses, want the 6 drawings", n)
	}
	run := 0
	for i, f := range sleep {
		run++
		if next := sleep[(i+1)%len(sleep)]; next == f {
			continue
		}
		if run <= fps/2 {
			t.Fatalf("sleep pose ending at frame %d is up for %d frames, want more than %d", i, run, fps/2)
		}
		run = 0
	}
}

// TestSleepKeepsItsPaceWhileAPanelIsOpen draws sleep at 60 window ticks a
// second, as while a panel is open, and sees the same pose as at 12 at every
// moment, with each pose up for more than half a second and a five-second loop.
func TestSleepKeepsItsPaceWhileAPanelIsOpen(t *testing.T) {
	sleep := Frames()["sleep"]
	at := func(elapsed time.Duration) *image.RGBA {
		return sleep[frameAt(pet.AnimTick(elapsed), len(sleep))]
	}
	const step = time.Second / 60
	var changes []time.Duration
	for elapsed := step; elapsed < 10*time.Second; elapsed += step {
		calm := elapsed.Truncate(time.Second / 12)
		if at(elapsed) != at(calm) {
			t.Fatalf("at %v the pose at 60 ticks a second differs from 12", elapsed)
		}
		if at(elapsed) != at(elapsed-step) {
			changes = append(changes, elapsed)
		}
	}
	for i := 1; i < len(changes); i++ {
		if gap := changes[i] - changes[i-1]; gap <= time.Second/2 {
			t.Fatalf("a sleep pose was up for %v at 60 ticks a second", gap)
		}
	}
	if loop := len(poses(sleep)); len(changes) <= loop || changes[loop]-changes[0] < 4*time.Second {
		t.Fatalf("sleep loop at 60 ticks a second: pose changes at %v", changes)
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
