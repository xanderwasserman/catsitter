package platform

// RestoreTarget is the window or process that should receive the keyboard
// again when a menu closes. It is empty once focus has already moved on,
// and empty when this process never took it.
func RestoreTarget(self, prev, current uint64) uint64 {
	if self == 0 || prev == 0 || prev == self || current != self {
		return 0
	}
	return prev
}
