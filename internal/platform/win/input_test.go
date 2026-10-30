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
