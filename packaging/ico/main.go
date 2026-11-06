// Command ico builds the Windows icon from the pixel-art icon.
//
//	go run ./packaging/ico packaging/icon.png packaging/icon.ico
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"golang.org/x/image/draw"

	"github.com/xanderwasserman/catsitter/internal/ico"
)

// sizes are the sizes Explorer and the Start menu ask for.
var sizes = []int{16, 24, 32, 48, 64, 256}

// artGrid is the cat's own pixel grid. icon.png draws it at 2x.
const artGrid = 32

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: ico SOURCE.png OUTPUT.ico")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}

func run(srcPath, outPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("%s: %w", srcPath, err)
	}
	return os.WriteFile(outPath, ico.Encode(Images(src)...), 0o644)
}

// Images is the art at each size in sizes.
func Images(art image.Image) []image.Image {
	out := make([]image.Image, len(sizes))
	for i, size := range sizes {
		out[i] = Resize(art, size)
	}
	return out
}

// Resize keeps hard pixel edges at whole multiples of artGrid and smooths
// the sizes in between.
func Resize(art image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	var scaler draw.Scaler = draw.CatmullRom
	if size%artGrid == 0 {
		scaler = draw.NearestNeighbor
	}
	scaler.Scale(dst, dst.Bounds(), art, art.Bounds(), draw.Src, nil)
	return dst
}
