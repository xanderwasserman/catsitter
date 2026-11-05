//go:build darwin

package mac

/*
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>

void csAskAccess(const char *title, const char *body, const char *open, const char *later);
void csCloseAccess(void);
void csToast(const char *text);
*/
import "C"
import "unsafe"

const (
	askTitle = "Let Catsitter nudge your Mac"
	askBody  = "When you step away, Catsitter moves the mouse by one pixel and taps Shift so your status stays active. macOS needs your OK for that.\n\n" +
		"Click Open Settings, then turn on Catsitter in the Accessibility list. Your screen stays awake in the meantime."
	askOpen     = "Open Settings"
	askLater    = "Not Now"
	grantedNote = "Thanks! Catsitter can nudge now."
)

// Prompt explains why Accessibility is needed and offers its settings page.
// It returns at once. The panel opens on the main thread.
func Prompt() {
	Post(func() {
		strs := []*C.char{C.CString(askTitle), C.CString(askBody), C.CString(askOpen), C.CString(askLater)}
		defer func() {
			for _, s := range strs {
				C.free(unsafe.Pointer(s))
			}
		}()
		C.csAskAccess(strs[0], strs[1], strs[2], strs[3])
	})
}

// Granted closes the explanation and shows a short note that nudges are on.
func Granted() {
	Post(func() {
		C.csCloseAccess()
		text := C.CString(grantedNote)
		defer C.free(unsafe.Pointer(text))
		C.csToast(text)
	})
}
