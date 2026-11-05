package tray

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
)

// icoFile packs the rimmed tray icon at each scale into an .ico.
// Each image is a 32-bit BGRA bitmap with an empty AND mask.
func icoFile(scales ...int) []byte {
	var images [][]byte
	var sizes []int
	for _, s := range scales {
		img := iconImage(s, true)
		images = append(images, icoBitmap(img))
		sizes = append(sizes, img.Bounds().Dx())
	}
	var buf bytes.Buffer
	le := binary.LittleEndian
	_ = binary.Write(&buf, le, [3]uint16{0, 1, uint16(len(images))})
	offset := 6 + 16*len(images)
	for i, data := range images {
		dim := byte(sizes[i])
		buf.Write([]byte{dim, dim, 0, 0})
		_ = binary.Write(&buf, le, [2]uint16{1, 32})
		_ = binary.Write(&buf, le, [2]uint32{uint32(len(data)), uint32(offset)})
		offset += len(data)
	}
	for _, data := range images {
		buf.Write(data)
	}
	return buf.Bytes()
}

// icoBitmap is a BITMAPINFOHEADER, the image bottom-up with straight alpha,
// and the AND mask. The header height counts both images.
func icoBitmap(img *image.RGBA) []byte {
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
			c := color.NRGBAModel.Convert(img.RGBAAt(x, y)).(color.NRGBA)
			buf.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	buf.Write(make([]byte, maskRow*h))
	return buf.Bytes()
}
