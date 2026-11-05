package tray

import (
	"os"
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

// loadIcon loads the file the way systray does.
func loadIcon(t *testing.T, data []byte, size int) bool {
	t.Helper()
	path := filepath.Join(t.TempDir(), "icon")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(path)
	const imageIcon, loadFromFile, defaultSize = 1, 0x10, 0x40
	h, _, _ := procLoadImage.Call(0, uintptr(unsafe.Pointer(name)), imageIcon,
		uintptr(size), uintptr(size), loadFromFile|defaultSize)
	if h != 0 {
		procDestroy.Call(h)
	}
	return h != 0
}

func TestWindowsLoadsTheTrayIcon(t *testing.T) {
	for _, size := range []int{0, 16, 24, 32} {
		if !loadIcon(t, menuBarIcon(), size) {
			t.Fatalf("LoadImage rejected the icon at %d", size)
		}
	}
	if loadIcon(t, iconPNG(1, false), 0) {
		t.Fatal("LoadImage accepted a bare PNG")
	}
}
