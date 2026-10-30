//go:build windows

package win

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	esContinuous      = 0x80000000
	esSystemRequired  = 0x00000001
	esDisplayRequired = 0x00000002

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
	procSetThreadExecState = kernel32.NewProc("SetThreadExecutionState")
)

type lastInputInfo struct {
	CbSize uint32
	DwTime uint32
}

// Inhibitor uses SetThreadExecutionState for system and display sleep.
type Inhibitor struct{ held bool }

func (i *Inhibitor) Acquire() error {
	if i.held {
		return nil
	}
	r, _, err := procSetThreadExecState.Call(uintptr(esContinuous | esSystemRequired | esDisplayRequired))
	if r == 0 {
		return fmt.Errorf("SetThreadExecutionState: %v", err)
	}
	i.held = true
	return nil
}

func (i *Inhibitor) Release() error {
	if !i.held {
		return nil
	}
	_, _, _ = procSetThreadExecState.Call(uintptr(esContinuous))
	i.held = false
	return nil
}

// IdleTime uses GetLastInputInfo.
func IdleTime() (time.Duration, error) {
	info := lastInputInfo{CbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	r, _, err := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0, fmt.Errorf("GetLastInputInfo: %v", err)
	}
	now, _, _ := procGetTickCount.Call()
	ms := uint32(now) - info.DwTime
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
	if count < 1 {
		count = 1
	}
	vk := uint16(vkMenu)
	if modifier == "cmd" {
		vk = vkLWin
	}
	if err := keyTap(vk, false); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	for i := 0; i < count; i++ {
		if err := keyTap(vkTab, false); err != nil {
			return err
		}
		time.Sleep(30 * time.Millisecond)
		if err := keyTap(vkTab, true); err != nil {
			return err
		}
		time.Sleep(40 * time.Millisecond)
	}
	return keyTap(vk, true)
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
