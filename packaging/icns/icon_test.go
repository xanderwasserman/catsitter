package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"testing"
)

type chunk struct {
	kind string
	data []byte
}

func parseICNS(t *testing.T, b []byte) []chunk {
	t.Helper()
	if len(b) < 8 || string(b[:4]) != "icns" {
		t.Fatal("missing icns header")
	}
	if n := binary.BigEndian.Uint32(b[4:8]); int(n) != len(b) {
		t.Fatalf("header length %d, file length %d", n, len(b))
	}
	var out []chunk
	for p := 8; p < len(b); {
		if p+8 > len(b) {
			t.Fatalf("truncated chunk at %d", p)
		}
		n := int(binary.BigEndian.Uint32(b[p+4 : p+8]))
		if n < 8 || p+n > len(b) {
			t.Fatalf("bad chunk length %d at %d", n, p)
		}
		out = append(out, chunk{kind: string(b[p : p+4]), data: b[p+8 : p+n]})
		p += n
	}
	return out
}

func loadPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// maxChannelDiff allows for floating-point differences between architectures.
const maxChannelDiff = 2

func closeEnough(t *testing.T, kind string, got, want image.Image) {
	t.Helper()
	if got.Bounds() != want.Bounds() {
		t.Fatalf("%s: bounds %v, want %v", kind, got.Bounds(), want.Bounds())
	}
	b := got.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r1, g1, b1, a1 := got.At(x, y).RGBA()
			r2, g2, b2, a2 := want.At(x, y).RGBA()
			for _, d := range [][2]uint32{{r1, r2}, {g1, g2}, {b1, b2}, {a1, a2}} {
				hi, lo := d[0]>>8, d[1]>>8
				if hi < lo {
					hi, lo = lo, hi
				}
				if hi-lo > maxChannelDiff {
					t.Fatalf("%s: pixel (%d,%d) differs; run go run ./packaging/icns packaging/icon.png packaging/icon.icns", kind, x, y)
				}
			}
		}
	}
}

func TestCommittedIconMatchesSource(t *testing.T) {
	icon := Compose(loadPNG(t, "../icon.png"))
	b, err := os.ReadFile("../icon.icns")
	if err != nil {
		t.Fatal(err)
	}
	chunks := parseICNS(t, b)
	if len(chunks) != len(reps) {
		t.Fatalf("got %d images, want %d", len(chunks), len(reps))
	}
	for i, r := range reps {
		c := chunks[i]
		if c.kind != r.kind {
			t.Fatalf("image %d is %s, want %s", i, c.kind, r.kind)
		}
		got, err := png.Decode(bytes.NewReader(c.data))
		if err != nil {
			t.Fatalf("%s: %v", c.kind, err)
		}
		closeEnough(t, c.kind, got, Resize(icon, r.size))
	}
}

func TestComposeCentresArtOnPlate(t *testing.T) {
	icon := Compose(loadPNG(t, "../icon.png"))
	if _, _, _, a := icon.At(10, 10).RGBA(); a != 0 {
		t.Fatal("corner outside the plate is not transparent")
	}
	if _, _, _, a := icon.At(plateInset+plateRadius, plateInset+4).RGBA(); a != 0xffff {
		t.Fatal("plate edge is not opaque")
	}
	if _, _, _, a := icon.At(canvas/2, canvas/2).RGBA(); a != 0xffff {
		t.Fatal("centre is not opaque")
	}
}

func TestEncodeLengths(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, Compose(image.NewRGBA(image.Rect(0, 0, 64, 64)))); err != nil {
		t.Fatal(err)
	}
	chunks := parseICNS(t, buf.Bytes())
	for i, c := range chunks {
		cfg, err := png.DecodeConfig(bytes.NewReader(c.data))
		if err != nil {
			t.Fatalf("%s: %v", c.kind, err)
		}
		if cfg.Width != reps[i].size || cfg.Height != reps[i].size {
			t.Fatalf("%s is %dx%d, want %d", c.kind, cfg.Width, cfg.Height, reps[i].size)
		}
	}
}
