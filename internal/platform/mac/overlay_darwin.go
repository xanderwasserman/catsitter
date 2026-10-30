//go:build darwin

package mac

/*
#cgo LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>

static int csAccessory(void) {
	NSApplication *app = [NSApplication sharedApplication];
	[app setActivationPolicy:NSApplicationActivationPolicyAccessory];
	NSWindow *win = [app keyWindow];
	if (win == nil) {
		NSArray *wins = [app windows];
		if ([wins count] == 0) {
			return -1;
		}
		win = [wins objectAtIndex:0];
	}
	[win setCollectionBehavior:
		(NSWindowCollectionBehaviorIgnoresCycle |
		 NSWindowCollectionBehaviorTransient |
		 NSWindowCollectionBehaviorStationary)];
	return 0;
}
*/
import "C"
import "fmt"

// ExcludeFromSwitcher hides Catsitter from the Dock and Cmd+Tab.
func ExcludeFromSwitcher() error {
	if C.csAccessory() != 0 {
		return fmt.Errorf("the pet window is not open yet")
	}
	return nil
}
