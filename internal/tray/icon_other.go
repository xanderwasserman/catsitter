//go:build !darwin

package tray

func menuBarIcon() []byte { return iconPNG(1, false) }
