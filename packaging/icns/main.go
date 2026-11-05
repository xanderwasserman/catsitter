// Command icns builds the macOS app icon from the pixel-art icon.
//
//	go run ./packaging/icns packaging/icon.png packaging/icon.icns
package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: icns SOURCE.png OUTPUT.icns")
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
	var out bytes.Buffer
	if err := Encode(&out, Compose(src)); err != nil {
		return err
	}
	return os.WriteFile(outPath, out.Bytes(), 0o644)
}
