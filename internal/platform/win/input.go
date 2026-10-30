package win

import "unsafe"

// Mouse and keyboard INPUT layouts for 64-bit Windows.
// The union starts at offset 8. MOUSEINPUT is 32 bytes, so INPUT is 40.
// KEYBDINPUT is shorter; tail pads the Go struct to the same size so
// SendInput's cbSize argument matches.

type MouseInput struct {
	Type      uint32  // 0
	Pad0      uint32  // 4
	DX        int32   // 8
	DY        int32   // 12
	MouseData uint32  // 16
	Flags     uint32  // 20
	Time      uint32  // 24
	Pad1      uint32  // 28
	Extra     uintptr // 32
}

type KeyInput struct {
	Type  uint32  // 0
	Pad0  uint32  // 4
	VK    uint16  // 8
	Scan  uint16  // 10
	Flags uint32  // 12
	Time  uint32  // 16
	Pad1  uint32  // 20
	Extra uintptr // 24
	Tail  uint64  // 32
}

func InputBytes() int {
	return int(unsafe.Sizeof(MouseInput{}))
}

func KeyInputBytes() int {
	return int(unsafe.Sizeof(KeyInput{}))
}

// IdleMillis is the elapsed time between two 32-bit tick counts.
// GetLastInputInfo stamps dwTime from the same clock as GetTickCount,
// so a uint32 subtraction stays correct when the clock wraps.
func IdleMillis(now, last uint32) uint32 {
	return now - last
}
