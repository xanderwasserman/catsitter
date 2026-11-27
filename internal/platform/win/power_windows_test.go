package win

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestReasonContextLayout64(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("layout is specified for 64-bit Windows")
	}
	var c reasonContext
	if unsafe.Sizeof(c) != 32 || unsafe.Offsetof(c.Reason) != 8 {
		t.Fatalf("size %d reason at %d", unsafe.Sizeof(c), unsafe.Offsetof(c.Reason))
	}
}

// onThread runs f on a fresh locked OS thread. The thread exits afterwards,
// so two calls never share one.
func onThread(f func() error) error {
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		done <- f()
	}()
	return <-done
}

func TestInhibitorReleasesFromAnotherThread(t *testing.T) {
	p := newPowerInhibitor(powerAPI{})
	for range 3 {
		if err := onThread(p.Acquire); err != nil {
			t.Fatalf("acquire: %v", err)
		}
		if err := onThread(p.Release); err != nil {
			t.Fatalf("release: %v", err)
		}
		if p.handle != 0 || len(p.set) != 0 {
			t.Fatalf("still held after release: handle %#x set %v", p.handle, p.set)
		}
	}
}
