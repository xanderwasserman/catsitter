//go:build windows

package win

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/xanderwasserman/catsitter/internal/platform"
	"golang.org/x/sys/windows"
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseEventMove = 0x0001
	keyEventKeyUp  = 0x0002

	vkShift = 0x10
	vkMenu  = 0x12
	vkLWin  = 0x5B
	vkTab   = 0x09
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procSendInput          = user32.NewProc("SendInput")
	procGetLastInputInfo   = user32.NewProc("GetLastInputInfo")
	procGetTickCount       = kernel32.NewProc("GetTickCount")
	procPowerCreateRequest = kernel32.NewProc("PowerCreateRequest")
	procPowerSetRequest    = kernel32.NewProc("PowerSetRequest")
	procPowerClearRequest  = kernel32.NewProc("PowerClearRequest")
)

type lastInputInfo struct {
	CbSize uint32
	DwTime uint32
}

// NewInhibitor keeps the system and display awake with a power request.
func NewInhibitor() platform.Inhibitor {
	return newPowerInhibitor(powerAPI{})
}

const (
	powerRequestContextVersion      = 0
	powerRequestContextSimpleString = 0x1
)

// reasonContext is REASON_CONTEXT with a simple reason string. pad covers
// the larger detailed-string member of the union; the size matches on 64-bit
// Windows only.
type reasonContext struct {
	Version uint32
	Flags   uint32
	Reason  *uint16
	pad     [2]uintptr
}

type powerAPI struct{}

func (powerAPI) create() (uintptr, error) {
	reason, err := windows.UTF16PtrFromString("Catsitter keeps the computer awake")
	if err != nil {
		return 0, err
	}
	ctx := reasonContext{Version: powerRequestContextVersion, Flags: powerRequestContextSimpleString, Reason: reason}
	h, _, err := procPowerCreateRequest.Call(uintptr(unsafe.Pointer(&ctx)))
	if windows.Handle(h) == windows.InvalidHandle || h == 0 {
		return 0, err
	}
	return h, nil
}

func (powerAPI) set(h uintptr, kind powerKind) error {
	if r, _, err := procPowerSetRequest.Call(h, uintptr(kind)); r == 0 {
		return err
	}
	return nil
}

func (powerAPI) clear(h uintptr, kind powerKind) error {
	if r, _, err := procPowerClearRequest.Call(h, uintptr(kind)); r == 0 {
		return err
	}
	return nil
}

func (powerAPI) close(h uintptr) error {
	return windows.CloseHandle(windows.Handle(h))
}

// IdleTime uses GetLastInputInfo.
func IdleTime() (time.Duration, error) {
	info := lastInputInfo{CbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	r, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0, fmt.Errorf("GetLastInputInfo: %v", err)
	}
	now, _, _ := procGetTickCount.Call()
	// dwTime and GetTickCount share a 32-bit clock. The subtraction wraps with it.
	ms := IdleMillis(uint32(now), info.DwTime)
	return time.Duration(ms) * time.Millisecond, nil
}

// Nudger posts SendInput events.
type Nudger struct{}

func (Nudger) Jiggle(pixels int) error {
	if pixels < 1 {
		pixels = 1
	}
	if err := mouseMove(int32(pixels), 0); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)
	return mouseMove(int32(-pixels), 0)
}

func (Nudger) TapShift() error {
	if err := keyTap(vkShift, false); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)
	return keyTap(vkShift, true)
}

func (Nudger) SwitchWindows(modifier string, count int) error {
	vk := uint16(vkMenu)
	if modifier == "cmd" {
		vk = vkLWin
	}
	return platform.HoldAndTap(
		func() error { return keyTap(vk, false) },
		func() error { return keyTap(vk, true) },
		func() error { return keyTap(vkTab, false) },
		func() error { return keyTap(vkTab, true) },
		count,
		func() { time.Sleep(20 * time.Millisecond) },
		func() { time.Sleep(30 * time.Millisecond) },
		func() { time.Sleep(40 * time.Millisecond) },
	)
}

func mouseMove(dx, dy int32) error {
	in := MouseInput{Type: inputMouse, DX: dx, DY: dy, Flags: mouseEventMove}
	return sendInput(unsafe.Pointer(&in))
}

func keyTap(vk uint16, up bool) error {
	in := KeyInput{Type: inputKeyboard, VK: vk}
	if up {
		in.Flags = keyEventKeyUp
	}
	return sendInput(unsafe.Pointer(&in))
}

func sendInput(in unsafe.Pointer) error {
	r, _, err := procSendInput.Call(1, uintptr(in), uintptr(InputBytes()))
	if r != 1 {
		return fmt.Errorf("SendInput: %v", err)
	}
	return nil
}
