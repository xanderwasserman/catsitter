//go:build !darwin && !windows

package tray

func menuBarIcon() []byte { return iconPNG(1, false) }

// removeOnStop is off: the item goes with the process's D-Bus connection,
// and systray's shutdown closes the shared session bus the inhibitor still needs.
const removeOnStop = false
