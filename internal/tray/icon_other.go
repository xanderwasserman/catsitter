//go:build !darwin && !windows

package tray

func menuBarIcon() []byte { return iconPNG(1, false) }
