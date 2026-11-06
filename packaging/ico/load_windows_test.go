package main

import (
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32        = windows.NewLazySystemDLL("user32.dll")
	procLoadImage = user32.NewProc("LoadImageW")
	procDestroy   = user32.NewProc("DestroyIcon")
)

func TestWindowsLoadsTheIcon(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("..", "icon.ico"))
	if err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(path)
	const imageIcon, loadFromFile = 1, 0x10
	for _, size := range sizes {
		h, _, _ := procLoadImage.Call(0, uintptr(unsafe.Pointer(name)), imageIcon,
			uintptr(size), uintptr(size), loadFromFile)
		if h == 0 {
			t.Fatalf("LoadImage rejected the icon at %d px", size)
		}
		procDestroy.Call(h)
	}
}
