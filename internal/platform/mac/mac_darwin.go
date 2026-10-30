//go:build darwin

package mac

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation -framework CoreGraphics -framework ApplicationServices

#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <ApplicationServices/ApplicationServices.h>
#include <IOKit/pwr_mgt/IOPMLib.h>

static IOPMAssertionID csDisplay;
static IOPMAssertionID csSystem;
static int csHeld;

static int csAcquire(void) {
	IOReturn a, b;
	if (csHeld) {
		return 0;
	}
	CFStringRef name = CFStringCreateWithCString(NULL, "Catsitter is keeping watch", kCFStringEncodingUTF8);
	a = IOPMAssertionCreateWithName(CFSTR("PreventUserIdleDisplaySleep"), kIOPMAssertionLevelOn, name, &csDisplay);
	b = IOPMAssertionCreateWithName(CFSTR("PreventUserIdleSystemSleep"), kIOPMAssertionLevelOn, name, &csSystem);
	CFRelease(name);
	if (a != kIOReturnSuccess || b != kIOReturnSuccess) {
		if (a == kIOReturnSuccess) IOPMAssertionRelease(csDisplay);
		if (b == kIOReturnSuccess) IOPMAssertionRelease(csSystem);
		return -1;
	}
	csHeld = 1;
	return 0;
}

static void csRelease(void) {
	if (!csHeld) {
		return;
	}
	IOPMAssertionRelease(csDisplay);
	IOPMAssertionRelease(csSystem);
	csHeld = 0;
}

static double csIdleSeconds(void) {
	return CGEventSourceSecondsSinceLastEventType(kCGEventSourceStateHIDSystemState, kCGAnyInputEventType);
}

static int csTrusted(void) {
	return AXIsProcessTrusted() ? 1 : 0;
}

static void csMouse(double *x, double *y) {
	CGEventRef e = CGEventCreate(NULL);
	CGPoint p = CGEventGetLocation(e);
	CFRelease(e);
	*x = p.x;
	*y = p.y;
}

static void csMove(double x, double y) {
	CGEventRef e = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, CGPointMake(x, y), kCGMouseButtonLeft);
	CGEventPost(kCGHIDEventTap, e);
	CFRelease(e);
}

static void csKey(unsigned short code, int down, unsigned long long flags) {
	CGEventRef e = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)code, down ? true : false);
	if (flags) {
		CGEventSetFlags(e, (CGEventFlags)flags);
	}
	CGEventPost(kCGHIDEventTap, e);
	CFRelease(e);
}
*/
import "C"
import (
	"fmt"
	"os/exec"
	"time"
)

// Inhibitor holds IOPM assertions for idle display sleep and idle system sleep.
type Inhibitor struct{}

func (Inhibitor) Acquire() error {
	if C.csAcquire() != 0 {
		return fmt.Errorf("IOPMAssertionCreateWithName failed")
	}
	return nil
}

func (Inhibitor) Release() error {
	C.csRelease()
	return nil
}

// IdleTime reads HID idle time. It does not require Accessibility permission.
func IdleTime() (time.Duration, error) {
	secs := float64(C.csIdleSeconds())
	if secs < 0 {
		secs = 0
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// Trusted reports whether Accessibility permission has been granted.
func Trusted() bool { return C.csTrusted() == 1 }

// Prompt explains the missing permission and opens the Accessibility pane.
func Prompt() {
	fmt.Println(`Catsitter can keep the display awake, but macOS is hiding the keyboard and mouse.

To allow the jiggle and the key taps, open System Settings → Privacy & Security → Accessibility and turn Catsitter on. The sleep inhibitor stays on in the meantime, and the other actions start on their own once you allow them.`)
	urls := []string{
		"x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?Privacy_Accessibility",
		"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility",
	}
	for _, u := range urls {
		if err := exec.Command("open", u).Run(); err == nil {
			return
		}
	}
}

const (
	vkShift   = 0x38
	vkOption  = 0x3A
	vkCommand = 0x37
	vkTab     = 0x30

	flagShift   = 0x00020000
	flagOption  = 0x00080000
	flagCommand = 0x00100000
)

// Nudger posts CoreGraphics events. Posting requires Accessibility permission.
type Nudger struct{}

func (Nudger) Jiggle(pixels int) error {
	if pixels < 1 {
		pixels = 1
	}
	var x, y C.double
	C.csMouse(&x, &y)
	C.csMove(x+C.double(pixels), y)
	time.Sleep(30 * time.Millisecond)
	C.csMove(x, y)
	return nil
}

func (Nudger) TapShift() error {
	tap(vkShift, flagShift)
	return nil
}

func (Nudger) SwitchWindows(modifier string, count int) error {
	if count < 1 {
		count = 1
	}
	code := C.ushort(vkOption)
	flags := C.ulonglong(flagOption)
	if modifier == "cmd" {
		code = vkCommand
		flags = flagCommand
	}
	C.csKey(code, 1, flags)
	time.Sleep(20 * time.Millisecond)
	for i := 0; i < count; i++ {
		C.csKey(vkTab, 1, flags)
		time.Sleep(30 * time.Millisecond)
		C.csKey(vkTab, 0, flags)
		time.Sleep(40 * time.Millisecond)
	}
	C.csKey(code, 0, 0)
	return nil
}

func tap(code C.ushort, flags C.ulonglong) {
	C.csKey(code, 1, flags)
	time.Sleep(30 * time.Millisecond)
	C.csKey(code, 0, 0)
}
