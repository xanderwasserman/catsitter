#import <AppKit/AppKit.h>

// csAccessory marks every window as an accessory that the switcher skips.
// The collection behavior is merged with the window's current value.
int csAccessory(void) {
	NSApplication *app = [NSApplication sharedApplication];
	if (app == nil) {
		return -1;
	}
	[app setActivationPolicy:NSApplicationActivationPolicyAccessory];

	NSArray *wins = [app windows];
	if ([wins count] == 0) {
		return -1;
	}
	NSWindowCollectionBehavior extra =
		NSWindowCollectionBehaviorIgnoresCycle |
		NSWindowCollectionBehaviorTransient |
		NSWindowCollectionBehaviorStationary;
	NSUInteger n = [wins count];
	for (NSUInteger i = 0; i < n; i++) {
		NSWindow *win = [wins objectAtIndex:i];
		NSWindowCollectionBehavior next = [win collectionBehavior] | extra;
		if (next != [win collectionBehavior]) {
			[win setCollectionBehavior:next];
		}
	}
	return 0;
}
