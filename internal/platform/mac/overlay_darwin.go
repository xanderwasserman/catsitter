//go:build darwin

package mac

/*
#cgo LDFLAGS: -framework AppKit

int csAccessory(void);
*/
import "C"
import "fmt"

// ExcludeFromSwitcher hides Catsitter from the Dock and the app switcher.
func ExcludeFromSwitcher() error {
	if C.csAccessory() != 0 {
		return fmt.Errorf("the pet window is not open yet")
	}
	return nil
}
