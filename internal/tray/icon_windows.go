package tray

// menuBarIcon is an .ico, the only format systray loads on Windows.
// LoadImage picks the system icon size, 32 px at 100% scaling up to 64 px
// at 200%, and the taskbar shows it at half that. The rim keeps the dark
// outline readable on a dark taskbar.
func menuBarIcon() []byte { return icoFile(1, 2, 3, 4) }
