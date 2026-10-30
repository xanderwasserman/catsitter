package win

import (
	"testing"
	"unsafe"
)

func TestInputLayout64(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("layout is specified for 64-bit Windows")
	}
	if InputBytes() != 40 || KeyInputBytes() != 40 {
		t.Fatalf("mouse %d key %d", InputBytes(), KeyInputBytes())
	}
	var m MouseInput
	var k KeyInput
	if unsafe.Offsetof(m.DX) != 8 || unsafe.Offsetof(m.Extra) != 32 {
		t.Fatalf("mouse offsets dx=%d extra=%d", unsafe.Offsetof(m.DX), unsafe.Offsetof(m.Extra))
	}
	if unsafe.Offsetof(k.VK) != 8 || unsafe.Offsetof(k.Extra) != 24 {
		t.Fatalf("key offsets vk=%d extra=%d", unsafe.Offsetof(k.VK), unsafe.Offsetof(k.Extra))
	}
}

func TestIdleMillisWraps(t *testing.T) {
	if IdleMillis(100, 40) != 60 {
		t.Fatalf("plain delta %d", IdleMillis(100, 40))
	}
	// last is just before the 32-bit clock wraps; now is just after.
	if IdleMillis(10, 0xFFFFFFF0) != 26 {
		t.Fatalf("wrapped delta %d", IdleMillis(10, 0xFFFFFFF0))
	}
}
