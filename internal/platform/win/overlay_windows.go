//go:build windows

package win

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wsExToolWindow  = 0x00000080
	wsExAppWindow   = 0x00040000
	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
)

var (
	procFindWindowW       = user32.NewProc("FindWindowW")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
)

// ExcludeFromSwitcher gives the pet window the tool-window style so the
// taskbar and Alt+Tab leave it out.
func ExcludeFromSwitcher(title string) error {
	name, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(name)))
	if hwnd == 0 {
		return fmt.Errorf("window %q was not found", title)
	}
	gwlExStyle := int32(-20)
	idx := uintptr(gwlExStyle)
	style, _, e := procGetWindowLongPtrW.Call(hwnd, idx)
	if style == 0 && errno(e) != 0 {
		return fmt.Errorf("GetWindowLongPtr: %w", e)
	}
	next := style | wsExToolWindow
	next &^= wsExAppWindow
	if next != style {
		prev, _, e := procSetWindowLongPtrW.Call(hwnd, idx, next)
		if prev == 0 && errno(e) != 0 {
			return fmt.Errorf("SetWindowLongPtr: %w", e)
		}
		flags := uintptr(swpNoSize | swpNoMove | swpNoZOrder | swpNoActivate | swpFrameChanged)
		procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, flags)
	}
	return nil
}

func errno(e error) syscall.Errno {
	n, _ := e.(syscall.Errno)
	return n
}
