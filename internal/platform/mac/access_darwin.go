//go:build darwin

package mac

/*
#cgo LDFLAGS: -framework AppKit -framework ApplicationServices -framework Security
#include <stdlib.h>

void csAskAccess(const char *window, const char *title, const char *body, const char *open, const char *later);
void csCloseAccess(void);
int csSignedAsBundle(void);
void csToast(const char *text);
*/
import "C"
import "unsafe"

const (
	askWindow = "Allow Accessibility"
	askTitle  = "Let Catsitter nudge your Mac"
	askBody   = "When you step away, Catsitter moves the mouse by one pixel and taps Shift so your status stays active. macOS needs your OK for that.\n\n" +
		"Click Open Settings. If macOS then asks, choose Open System Settings. Turn on Catsitter in the Accessibility list. Your screen stays awake in the meantime."
	unsignedNote = "\n\nThis copy was installed without a signature, so macOS forgets the permission. Run the Catsitter install command in Terminal once to fix that."
	askOpen      = "Open Settings"
	askLater     = "Not Now"
	grantedNote  = "Thanks! Catsitter can nudge now."
	reopenNote   = "Thanks! Quit and reopen Catsitter to start nudging."
)

// Prompt explains why Accessibility is needed and offers its settings page.
// It returns at once. The panel opens on the main thread.
func Prompt() {
	Post(func() {
		strs := []*C.char{C.CString(askWindow), C.CString(askTitle), C.CString(askText()), C.CString(askOpen), C.CString(askLater)}
		defer func() {
			for _, s := range strs {
				C.free(unsafe.Pointer(s))
			}
		}()
		C.csAskAccess(strs[0], strs[1], strs[2], strs[3], strs[4])
	})
}

// askText is the explanation, with a fix for copies the current installer
// did not sign.
func askText() string {
	if C.csSignedAsBundle() == 0 {
		return askBody + unsignedNote
	}
	return askBody
}

// Granted closes the explanation and shows a short note that nudges are on.
func Granted() { note(grantedNote) }

// Reopen closes the explanation and asks the user to quit and reopen Catsitter.
func Reopen() { note(reopenNote) }

func note(s string) {
	Post(func() {
		C.csCloseAccess()
		text := C.CString(s)
		defer C.free(unsafe.Pointer(text))
		C.csToast(text)
	})
}
