package tray

// menuBarIcon is drawn at 2x for Retina menu bars. The rim keeps the dark
// outline readable on a dark menu bar.
func menuBarIcon() []byte { return iconPNG(2, true) }
