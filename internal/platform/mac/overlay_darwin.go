//go:build darwin

package mac

/*
#cgo LDFLAGS: -framework AppKit

int csAccessory(void);
void csNoteFront(void);
void csFrontPIDs(int *selfPID, int *frontPID, int *prevPID);
void csActivate(int pid);
*/
import "C"
import (
	"fmt"

	"github.com/xanderwasserman/catsitter/internal/platform"
)

// ExcludeFromSwitcher hides Catsitter from the Dock and the app switcher.
func ExcludeFromSwitcher() error {
	if C.csAccessory() != 0 {
		return fmt.Errorf("the pet window is not open yet")
	}
	return nil
}

// Front remembers which app had the keyboard before this one.
type Front struct {
	self int
	prev int
}

// Note records the frontmost app while it is not this process.
func (f *Front) Note() {
	var self, front, prev C.int
	C.csNoteFront()
	C.csFrontPIDs(&self, &front, &prev)
	f.self = int(self)
	if front != 0 && int(front) != f.self {
		f.prev = int(front)
	} else if prev != 0 {
		f.prev = int(prev)
	}
}

// Restore gives the keyboard back when this process still holds it.
func (f *Front) Restore() {
	var self, front, prev C.int
	C.csFrontPIDs(&self, &front, &prev)
	f.self = int(self)
	if int(prev) != 0 {
		f.prev = int(prev)
	}
	target := platform.RestoreTarget(uint64(f.self), uint64(f.prev), uint64(front))
	if target == 0 {
		return
	}
	C.csActivate(C.int(target))
}
