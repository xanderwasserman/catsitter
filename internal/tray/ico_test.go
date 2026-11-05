package tray

import (
	"encoding/binary"
	"image/color"
	"testing"
)

func TestIcoHoldsEachScale(t *testing.T) {
	ico := icoFile(1, 2, 3)
	le := binary.LittleEndian
	if le.Uint16(ico[0:]) != 0 || le.Uint16(ico[2:]) != 1 || le.Uint16(ico[4:]) != 3 {
		t.Fatalf("header % x", ico[:6])
	}
	for i, scale := range []int{1, 2, 3} {
		entry := ico[6+16*i:]
		size := 16 * scale
		if int(entry[0]) != size || int(entry[1]) != size || le.Uint16(entry[6:]) != 32 {
			t.Fatalf("entry %d: % x", i, entry[:16])
		}
		n, off := int(le.Uint32(entry[8:])), int(le.Uint32(entry[12:]))
		mask := (size + 31) / 32 * 4 * size
		if n != 40+size*size*4+mask || off+n > len(ico) {
			t.Fatalf("entry %d: %d bytes at %d", i, n, off)
		}
		dib := ico[off : off+n]
		if le.Uint32(dib) != 40 || int(int32(le.Uint32(dib[4:]))) != size || int(int32(le.Uint32(dib[8:]))) != 2*size {
			t.Fatalf("entry %d header % x", i, dib[:12])
		}
		want := iconImage(scale, true)
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				p := dib[40+((size-1-y)*size+x)*4:]
				got := color.NRGBA{p[2], p[1], p[0], p[3]}
				if exp := color.NRGBAModel.Convert(want.RGBAAt(x, y)); got != exp {
					t.Fatalf("scale %d (%d,%d): %v want %v", scale, x, y, got, exp)
				}
			}
		}
	}
}

func TestIcoDim(t *testing.T) {
	if icoDim(16) != 16 || icoDim(256) != 0 {
		t.Fatalf("16 -> %d, 256 -> %d", icoDim(16), icoDim(256))
	}
	defer func() {
		if recover() == nil {
			t.Fatal("257 px did not panic")
		}
	}()
	icoDim(257)
}
