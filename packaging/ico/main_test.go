package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// maxChannelDiff allows for floating-point differences between architectures.
const maxChannelDiff = 2

// decode reads every image back out of an .ico written by ico.Encode.
func decode(t *testing.T, file []byte) []image.Image {
	t.Helper()
	le := binary.LittleEndian
	var out []image.Image
	for i := 0; i < int(le.Uint16(file[4:])); i++ {
		e := file[6+16*i:]
		n, off := int(le.Uint32(e[8:])), int(le.Uint32(e[12:]))
		data := file[off : off+n]
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			out = append(out, img)
			continue
		}
		size := int(int32(le.Uint32(data[4:])))
		img := image.NewNRGBA(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				p := data[40+((size-1-y)*size+x)*4:]
				img.SetNRGBA(x, y, color.NRGBA{p[2], p[1], p[0], p[3]})
			}
		}
		out = append(out, img)
	}
	return out
}

func TestCommittedIconMatchesSource(t *testing.T) {
	f, err := os.Open("../icon.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.ReadFile("../icon.ico")
	if err != nil {
		t.Fatal(err)
	}
	got, want := decode(t, file), Images(src)
	if len(got) != len(want) {
		t.Fatalf("icon.ico holds %d images, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Bounds() != want[i].Bounds() {
			t.Fatalf("image %d is %v, want %v", i, got[i].Bounds(), want[i].Bounds())
		}
		b := want[i].Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				g := color.NRGBAModel.Convert(got[i].At(x, y)).(color.NRGBA)
				w := color.NRGBAModel.Convert(want[i].At(x, y)).(color.NRGBA)
				for _, d := range [][2]uint8{{g.R, w.R}, {g.G, w.G}, {g.B, w.B}, {g.A, w.A}} {
					if int(d[0])-int(d[1]) > maxChannelDiff || int(d[1])-int(d[0]) > maxChannelDiff {
						t.Fatalf("%d px differs at (%d,%d); run go run ./packaging/ico packaging/icon.png packaging/icon.ico", b.Dx(), x, y)
					}
				}
			}
		}
	}
}

func TestResizeKeepsPixelEdgesOnTheGrid(t *testing.T) {
	art := image.NewRGBA(image.Rect(0, 0, 2*artGrid, 2*artGrid))
	for y := 0; y < 2*artGrid; y++ {
		for x := 0; x < 2*artGrid; x++ {
			if (x/2+y/2)%2 == 0 {
				art.Pix[art.PixOffset(x, y)+3] = 255
			}
		}
	}
	for _, size := range []int{32, 64, 256} {
		img := Resize(art, size)
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if a := img.RGBAAt(x, y).A; a != 0 && a != 255 {
					t.Fatalf("%d px has a blended pixel at (%d,%d)", size, x, y)
				}
			}
		}
	}
}
