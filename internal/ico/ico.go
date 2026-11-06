// Package ico writes Windows .ico files.
package ico

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// pngFrom is the smallest size stored as PNG rather than as a bitmap.
// Windows reads PNG entries from Vista on, and they keep 256 px small.
const pngFrom = 256

// Encode packs square images into an .ico, in the order given. Each image
// below 256 px is a 32-bit BGRA bitmap with an empty AND mask.
func Encode(images ...image.Image) []byte {
	var buf bytes.Buffer
	le := binary.LittleEndian
	_ = binary.Write(&buf, le, [3]uint16{0, 1, uint16(len(images))})
	offset := 6 + 16*len(images)
	var data [][]byte
	for _, img := range images {
		bits := entry(img)
		dim := Dim(img.Bounds().Dx())
		buf.Write([]byte{dim, dim, 0, 0})
		_ = binary.Write(&buf, le, [2]uint16{1, 32})
		_ = binary.Write(&buf, le, [2]uint32{uint32(len(bits)), uint32(offset)})
		offset += len(bits)
		data = append(data, bits)
	}
	for _, bits := range data {
		buf.Write(bits)
	}
	return buf.Bytes()
}

// Dim is a directory entry's size byte, where 0 means 256.
func Dim(px int) byte {
	if px < 1 || px > 256 {
		panic(fmt.Sprintf("ico images are 1 to 256 px, not %d", px))
	}
	return byte(px % 256)
}

func entry(img image.Image) []byte {
	if img.Bounds().Dx() >= pngFrom {
		var buf bytes.Buffer
		_ = png.Encode(&buf, img)
		return buf.Bytes()
	}
	return bitmap(img)
}

// bitmap is a BITMAPINFOHEADER, the image bottom-up with straight alpha,
// and the AND mask. The header height counts both images.
func bitmap(img image.Image) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	maskRow := (w + 31) / 32 * 4
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, struct {
		Size          uint32
		Width, Height int32
		Planes, Bits  uint16
		Compression   uint32
		ImageSize     uint32
		XPels, YPels  int32
		Used, Needed  uint32
	}{Size: 40, Width: int32(w), Height: int32(2 * h), Planes: 1, Bits: 32,
		ImageSize: uint32(w*h*4 + maskRow*h)})
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			buf.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	buf.Write(make([]byte, maskRow*h))
	return buf.Bytes()
}
