package cat

import (
	"bytes"
	"image"
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
	if sameImage(all["idle"][0], all["idle"][2]) {
		t.Fatal("blink frame matches open eyes")
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
