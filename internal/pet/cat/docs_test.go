package cat

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var docsDir = filepath.Join("..", "..", "..", "docs")

// previewed are the animations with a docs/<name>.png sheet and .gif.
var previewed = []string{"idle", "walk", "sleep", "paws"}

// readmeStrip is docs/cat.png: idle, walking, asleep and pawing at 5x.
func readmeStrip() *image.RGBA {
	const scale, step = 5, 180
	all := Frames()
	picks := []*image.RGBA{all["idle"][0], all["walk"][0], poses(all["sleep"])[2], all["paws"][0]}
	img := image.NewRGBA(image.Rect(0, 0, step*(len(picks)-1)+frameW*scale, frameH*scale))
	for i, f := range picks {
		drawScaled(img, f, i*step, scale)
	}
	return img
}

// frameSheet is docs/<name>.png: every pose of an animation at 8x, with a
// held frame drawn once.
func frameSheet(name string) *image.RGBA {
	const scale, gap = 8, 32
	frames := poses(Frames()[name])
	step := frameW*scale + gap
	img := image.NewRGBA(image.Rect(0, 0, step*len(frames)-gap, frameH*scale))
	for i, f := range frames {
		drawScaled(img, f, i*step, scale)
	}
	return img
}

// animation is docs/<name>.gif: an animation at 8x and the app's 6 frames a second.
func animation(name string) *gif.GIF {
	const scale = 8
	pal := make(color.Palette, len(palette))
	for i, c := range palette {
		pal[i] = c
	}
	anim := &gif.GIF{}
	for _, f := range Frames()[name] {
		big := image.NewRGBA(image.Rect(0, 0, frameW*scale, frameH*scale))
		drawScaled(big, f, 0, scale)
		frame := image.NewPaletted(big.Bounds(), pal)
		for y := 0; y < big.Bounds().Dy(); y++ {
			for x := 0; x < big.Bounds().Dx(); x++ {
				frame.Set(x, y, big.RGBAAt(x, y))
			}
		}
		anim.Image = append(anim.Image, frame)
		anim.Delay = append(anim.Delay, 17)
		anim.Disposal = append(anim.Disposal, gif.DisposalBackground)
	}
	return anim
}

func drawScaled(dst *image.RGBA, src *image.RGBA, x0, scale int) {
	for y := 0; y < frameH*scale; y++ {
		for x := 0; x < frameW*scale; x++ {
			dst.SetRGBA(x0+x, y, src.RGBAAt(x/scale, y/scale))
		}
	}
}

func TestDocsMatchTheArt(t *testing.T) {
	pictures := map[string]image.Image{"cat.png": readmeStrip()}
	for _, name := range previewed {
		pictures[name+".png"] = frameSheet(name)
	}
	if os.Getenv("UPDATE_SPRITES") == "1" {
		for name, img := range pictures {
			writePNG(t, filepath.Join(docsDir, name), img)
		}
		for _, name := range previewed {
			var buf bytes.Buffer
			if err := gif.EncodeAll(&buf, animation(name)); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(docsDir, name+".gif"), buf.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	for name, want := range pictures {
		b, err := os.ReadFile(filepath.Join(docsDir, name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		if !samePixels(got, want) {
			t.Fatalf("docs/%s does not match the pixel data (run with UPDATE_SPRITES=1)", name)
		}
	}
	for _, name := range previewed {
		b, err := os.ReadFile(filepath.Join(docsDir, name+".gif"))
		if err != nil {
			t.Fatal(err)
		}
		anim, err := gif.DecodeAll(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		want := animation(name).Image
		if len(anim.Image) != len(want) {
			t.Fatalf("docs/%s.gif has %d frames, want %d (run with UPDATE_SPRITES=1)", name, len(anim.Image), len(want))
		}
		for i := range want {
			if !samePixels(anim.Image[i], want[i]) {
				t.Fatalf("docs/%s.gif frame %d does not match the pixel data (run with UPDATE_SPRITES=1)", name, i)
			}
		}
	}
}
