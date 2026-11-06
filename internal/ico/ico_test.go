package ico

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func square(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	img.SetRGBA(0, 0, color.RGBA{200, 100, 50, 255})
	return img
}

func TestEncodeStoresBitmapsAndLargePNG(t *testing.T) {
	sizes := []int{16, 32, 256}
	var images []image.Image
	for _, s := range sizes {
		images = append(images, square(s))
	}
	file := Encode(images...)
	le := binary.LittleEndian
	if le.Uint16(file[0:]) != 0 || le.Uint16(file[2:]) != 1 || int(le.Uint16(file[4:])) != len(sizes) {
		t.Fatalf("header % x", file[:6])
	}
	for i, size := range sizes {
		e := file[6+16*i:]
		if e[0] != Dim(size) || e[1] != Dim(size) || le.Uint16(e[6:]) != 32 {
			t.Fatalf("entry %d: % x", i, e[:16])
		}
		n, off := int(le.Uint32(e[8:])), int(le.Uint32(e[12:]))
		if off+n > len(file) {
			t.Fatalf("entry %d: %d bytes at %d past the end", i, n, off)
		}
		data := file[off : off+n]
		if size >= pngFrom {
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil || img.Bounds().Dx() != size {
				t.Fatalf("entry %d is not a %d px PNG: %v", i, size, err)
			}
			continue
		}
		mask := (size + 31) / 32 * 4 * size
		if n != 40+size*size*4+mask || le.Uint32(data) != 40 || int(int32(le.Uint32(data[8:]))) != 2*size {
			t.Fatalf("entry %d bitmap header % x, %d bytes", i, data[:12], n)
		}
		// The first image row is stored last.
		p := data[40+(size-1)*size*4:]
		if got := (color.NRGBA{p[2], p[1], p[0], p[3]}); got != (color.NRGBA{200, 100, 50, 255}) {
			t.Fatalf("entry %d top-left pixel %v", i, got)
		}
	}
}

func TestDim(t *testing.T) {
	if Dim(16) != 16 || Dim(256) != 0 {
		t.Fatalf("16 -> %d, 256 -> %d", Dim(16), Dim(256))
	}
	defer func() {
		if recover() == nil {
			t.Fatal("257 px did not panic")
		}
	}()
	Dim(257)
}
