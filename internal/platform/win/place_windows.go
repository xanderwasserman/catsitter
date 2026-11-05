//go:build windows

package win

import (
	"sync"
	"unsafe"

	"github.com/xanderwasserman/catsitter/internal/platform"
	"golang.org/x/sys/windows"
)

var (
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procGetDpiForWindow     = user32.NewProc("GetDpiForWindow")
	procIsWindow            = user32.NewProc("IsWindow")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
)

type cursorPoint struct {
	X int32
	Y int32
}

type rect struct {
	Left, Top, Right, Bottom int32
}

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

// Window places the pet window in physical desktop pixels, divided by the
// window's own DPI scale. Ebitengine's SetWindowPosition counts from the top
// left of the window's current monitor, so a position from GetCursorPos,
// which counts from the primary monitor, lands one monitor offset away.
// All calls must run on the thread that owns the window.
type Window struct {
	title string
	hwnd  uintptr
}

// NewWindow finds the window by its title when first used.
func NewWindow(title string) *Window {
	return &Window{title: title}
}

// Placer exposes the window as the pet's placement hooks.
func (w *Window) Placer() *platform.Placer {
	return &platform.Placer{
		Scale:    w.Scale,
		Position: w.Position,
		Move:     w.Move,
		Pointer:  w.Pointer,
		Areas:    w.Areas,
	}
}

func (w *Window) handle() uintptr {
	if w.hwnd != 0 {
		if ok, _, _ := procIsWindow.Call(w.hwnd); ok != 0 {
			return w.hwnd
		}
	}
	w.hwnd = findWindow(w.title)
	return w.hwnd
}

// Scale is the window's DPI over 96. It reads the window, not a cached monitor,
// so it changes on the frame the window crosses onto a monitor with another scale.
func (w *Window) Scale(fallback float64) float64 {
	if hwnd := w.handle(); hwnd != 0 && procGetDpiForWindow.Find() == nil {
		if dpi, _, _ := procGetDpiForWindow.Call(hwnd); dpi > 0 {
			return float64(dpi) / 96
		}
	}
	if fallback <= 0 {
		return 1
	}
	return fallback
}

func (w *Window) rect() (rect, bool) {
	hwnd := w.handle()
	if hwnd == 0 {
		return rect{}, false
	}
	var r rect
	ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r, ok != 0
}

func (w *Window) Position(fallback float64) (int, int, bool) {
	r, ok := w.rect()
	if !ok {
		return 0, 0, false
	}
	s := w.Scale(fallback)
	return platform.DIP(int(r.Left), s), platform.DIP(int(r.Top), s), true
}

// Move keeps the size, the topmost order and the keyboard focus.
func (w *Window) Move(x, y int, fallback float64) {
	r, ok := w.rect()
	if !ok {
		return
	}
	s := w.Scale(fallback)
	px, py := platform.Physical(x, s), platform.Physical(y, s)
	if int(r.Left) == px && int(r.Top) == py {
		return
	}
	flags := uintptr(swpNoSize | swpNoZOrder | swpNoActivate)
	procSetWindowPos.Call(w.handle(), 0, uintptr(int32(px)), uintptr(int32(py)), 0, 0, flags)
}

// Pointer reads GetCursorPos, which is in physical pixels for this
// per-monitor DPI aware process.
func (w *Window) Pointer(fallback float64) (int, int, bool) {
	var p cursorPoint
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0, false
	}
	s := w.Scale(fallback)
	return platform.DIP(int(p.X), s), platform.DIP(int(p.Y), s), true
}

// Areas lists each monitor's work area, which leaves out the taskbar.
func (w *Window) Areas(fallback float64) []platform.Rect {
	s := w.Scale(fallback)
	var out []platform.Rect
	for _, r := range workAreas() {
		out = append(out, platform.RectDIP(int(r.Left), int(r.Top), int(r.Right), int(r.Bottom), s))
	}
	return out
}

var (
	monitorsMu  sync.Mutex
	monitorsOut []rect
	monitorEnum = windows.NewCallback(func(hmon, _, _, _ uintptr) uintptr {
		info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		if ok, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&info))); ok != 0 {
			monitorsOut = append(monitorsOut, info.Work)
		}
		return 1
	})
)

func workAreas() []rect {
	monitorsMu.Lock()
	defer monitorsMu.Unlock()
	monitorsOut = nil
	procEnumDisplayMonitors.Call(0, 0, monitorEnum, 0)
	out := monitorsOut
	monitorsOut = nil
	return out
}
