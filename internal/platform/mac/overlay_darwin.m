#import <AppKit/AppKit.h>
#include <sys/types.h>

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

static pid_t csPrevPID;

// csFrontPIDs reports this process, the frontmost process, and the last
// frontmost process that was not this one.
void csFrontPIDs(int *selfPID, int *frontPID, int *prevPID) {
	*selfPID = 0;
	*frontPID = 0;
	*prevPID = (int)csPrevPID;
	NSRunningApplication *self = [NSRunningApplication currentApplication];
	NSRunningApplication *front = [[NSWorkspace sharedWorkspace] frontmostApplication];
	if (self != nil) {
		*selfPID = (int)[self processIdentifier];
	}
	if (front != nil) {
		*frontPID = (int)[front processIdentifier];
	}
}

// csNoteFront remembers the frontmost app while it is not this process.
void csNoteFront(void) {
	int selfPID = 0;
	int frontPID = 0;
	int prevPID = 0;
	csFrontPIDs(&selfPID, &frontPID, &prevPID);
	if (frontPID != 0 && frontPID != selfPID) {
		csPrevPID = (pid_t)frontPID;
	}
}

// csActivate brings that process forward.
void csActivate(int pid) {
	if (pid == 0) {
		return;
	}
	NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:(pid_t)pid];
	if (app == nil) {
		return;
	}
	[app activateWithOptions:NSApplicationActivateIgnoringOtherApps];
}
