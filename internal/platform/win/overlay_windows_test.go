package win

import (
	"runtime"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

// ownWindow creates a hidden top-level window on the calling thread.
// The caller destroys it on the same thread.
func ownWindow(t *testing.T, title string) uintptr {
	t.Helper()
	class, _ := windows.UTF16PtrFromString("CatsitterTest")
	module, _, _ := procGetModuleHandleW.Call(0)
	wc := wndClassEx{WndProc: procDefWindowProcW.Addr(), Instance: windows.Handle(module), ClassName: class}
	wc.Size = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	name, _ := windows.UTF16PtrFromString(title)
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(name)), 0, 0, 0, 10, 10, 0, 0, module, 0)
	if hwnd == 0 {
		t.Fatalf("CreateWindowEx: %v", err)
	}
	return hwnd
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      cursorPoint
	Private uint32
}

func pump() {
	var msg msg
	for {
		r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
		if r == 0 {
			return
		}
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func TestExcludeFromSwitcherOnTheOwningThread(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hwnd := ownWindow(t, "catsitter-owner-test")
	defer procDestroyWindow.Call(hwnd)
	if err := ExcludeFromSwitcher("catsitter-owner-test"); err != nil {
		t.Fatal(err)
	}
	gwlExStyle := int32(-20)
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle))
	if style&wsExToolWindow == 0 {
		t.Fatalf("ex style %#x", style)
	}
}

// The pet window's thread waits in Go code between frames. A style change
// from another thread blocks until the owner reads its messages.
func TestExcludeFromSwitcherWaitsForTheOwningThread(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hwnd := ownWindow(t, "catsitter-cross-test")
	defer procDestroyWindow.Call(hwnd)
	done := make(chan error, 1)
	go func() { done <- ExcludeFromSwitcher("catsitter-cross-test") }()
	select {
	case err := <-done:
		t.Fatalf("returned without the owner pumping: %v", err)
	case <-time.After(time.Second):
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		pump()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatal("still blocked after the owner pumped")
}
