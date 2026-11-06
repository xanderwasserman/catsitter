package tray

import (
	"image"

	"github.com/xanderwasserman/catsitter/internal/ico"
)

// icoFile packs the rimmed tray icon at each scale into an .ico.
func icoFile(scales ...int) []byte {
	var images []image.Image
	for _, s := range scales {
		images = append(images, iconImage(s, true))
	}
	return ico.Encode(images...)
}
