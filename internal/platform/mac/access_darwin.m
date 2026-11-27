#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Security/Security.h>

static void csActivateSelf(void) {
	if (@available(macOS 14, *)) {
		[NSApp activate];
	} else {
		[NSApp activateIgnoringOtherApps:YES];
	}
}

static NSString *const csAuthAlertApp = @"com.apple.accessibility.universalAccessAuthWarn";
static const int csAuthAlertChecks = 10;
static const int64_t csAuthAlertInterval = 200 * NSEC_PER_MSEC;

static void csOpenAccessPane(void) {
	NSURL *url = [NSURL URLWithString:@"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"];
	[[NSWorkspace sharedWorkspace] openURL:url];
}

// csAuthAlertShowing reports whether macOS's own Accessibility alert is on
// screen. Window owners are readable without extra permission.
static BOOL csAuthAlertShowing(void) {
	@autoreleasepool {
		NSMutableSet *pids = [NSMutableSet set];
		for (NSRunningApplication *app in [NSRunningApplication runningApplicationsWithBundleIdentifier:csAuthAlertApp]) {
			[pids addObject:@([app processIdentifier])];
		}
		if ([pids count] == 0) {
			return NO;
		}
		CFArrayRef windows = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly, kCGNullWindowID);
		if (windows == NULL) {
			return NO;
		}
		BOOL showing = NO;
		for (NSDictionary *info in (NSArray *)windows) {
			if ([pids containsObject:info[(id)kCGWindowOwnerPID]]) {
				showing = YES;
				break;
			}
		}
		CFRelease(windows);
		return showing;
	}
}

// csOpenPaneUnlessAlert leaves the way to Settings to macOS's alert when it
// appears, and opens the pane itself otherwise.
static void csOpenPaneUnlessAlert(int checks) {
	if (csAuthAlertShowing()) {
		return;
	}
	if (checks == 0) {
		csOpenAccessPane();
		return;
	}
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, csAuthAlertInterval), dispatch_get_main_queue(), ^{
		csOpenPaneUnlessAlert(checks - 1);
	});
}

// csRequestAccess asks macOS to list Catsitter under Accessibility. macOS
// shows its alert, which opens Settings, only while Catsitter is not listed
// yet, so the pane is opened directly when no alert appears.
static void csRequestAccess(void) {
	@autoreleasepool {
		NSDictionary *ask = @{(id)kAXTrustedCheckOptionPrompt: @YES};
		AXIsProcessTrustedWithOptions((CFDictionaryRef)ask);
	}
	csOpenPaneUnlessAlert(csAuthAlertChecks);
}

// csSignedAsBundle reports whether this app's code identity is its bundle
// identifier, which macOS needs to keep an Accessibility grant. A bare binary
// has no bundle identifier and counts as signed.
int csSignedAsBundle(void) {
	@autoreleasepool {
		NSString *bundle = [[NSBundle mainBundle] bundleIdentifier];
		if (bundle == nil) {
			return 1;
		}
		SecCodeRef code = NULL;
		if (SecCodeCopySelf(kSecCSDefaultFlags, &code) != errSecSuccess) {
			return 0;
		}
		SecStaticCodeRef staticCode = NULL;
		OSStatus status = SecCodeCopyStaticCode(code, kSecCSDefaultFlags, &staticCode);
		CFRelease(code);
		if (status != errSecSuccess) {
			return 0;
		}
		CFDictionaryRef info = NULL;
		status = SecCodeCopySigningInformation(staticCode, kSecCSDefaultFlags, &info);
		CFRelease(staticCode);
		if (status != errSecSuccess) {
			return 0;
		}
		NSString *identifier = ((NSDictionary *)info)[(id)kSecCodeInfoIdentifier];
		int same = [identifier isEqualToString:bundle] ? 1 : 0;
		CFRelease(info);
		return same;
	}
}

@interface CSAccessPanel : NSObject
@property (nonatomic, retain) NSPanel *panel;
@end

@implementation CSAccessPanel
- (void)openSettings:(id)sender {
	[self.panel orderOut:nil];
	csRequestAccess();
}
- (void)later:(id)sender {
	[self.panel orderOut:nil];
}
@end

static CSAccessPanel *csAccess;

static NSTextField *csLabel(const char *text, NSFont *font) {
	NSTextField *label = [NSTextField wrappingLabelWithString:@(text)];
	[label setFont:font];
	[label setPreferredMaxLayoutWidth:320];
	[label setSelectable:NO];
	return label;
}

static NSButton *csButton(const char *title, id target, SEL action, NSString *key) {
	NSButton *button = [NSButton buttonWithTitle:@(title) target:target action:action];
	[button setKeyEquivalent:key];
	return button;
}

static CSAccessPanel *csBuildAccessPanel(const char *window, const char *title, const char *body,
                                         const char *open, const char *later) {
	CSAccessPanel *access = [[CSAccessPanel alloc] init];

	NSImageView *icon = [NSImageView imageViewWithImage:[NSApp applicationIconImage]];
	[icon setTranslatesAutoresizingMaskIntoConstraints:NO];
	[[icon.widthAnchor constraintEqualToConstant:64] setActive:YES];
	[[icon.heightAnchor constraintEqualToConstant:64] setActive:YES];

	NSStackView *text = [NSStackView stackViewWithViews:@[
		csLabel(title, [NSFont boldSystemFontOfSize:13]),
		csLabel(body, [NSFont systemFontOfSize:12]),
	]];
	[text setOrientation:NSUserInterfaceLayoutOrientationVertical];
	[text setAlignment:NSLayoutAttributeLeading];
	[text setSpacing:8];

	NSStackView *top = [NSStackView stackViewWithViews:@[icon, text]];
	[top setAlignment:NSLayoutAttributeTop];
	[top setSpacing:16];

	NSStackView *buttons = [NSStackView stackViewWithViews:@[
		csButton(later, access, @selector(later:), @"\033"),
		csButton(open, access, @selector(openSettings:), @"\r"),
	]];

	NSStackView *all = [NSStackView stackViewWithViews:@[top, buttons]];
	[all setOrientation:NSUserInterfaceLayoutOrientationVertical];
	[all setAlignment:NSLayoutAttributeTrailing];
	[all setSpacing:20];
	[all setEdgeInsets:NSEdgeInsetsMake(20, 20, 20, 20)];

	// Non-activating, so the buttons work on the first click even when
	// macOS keeps another app in front.
	NSPanel *panel = [[[NSPanel alloc] initWithContentRect:NSMakeRect(0, 0, 420, 200)
	                                             styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
	                                                       NSWindowStyleMaskNonactivatingPanel
	                                               backing:NSBackingStoreBuffered
	                                                 defer:NO] autorelease];
	[panel setTitle:@(window)];
	[panel setReleasedWhenClosed:NO];
	[panel setHidesOnDeactivate:NO];
	[panel setLevel:NSFloatingWindowLevel];
	[panel setCollectionBehavior:NSWindowCollectionBehaviorMoveToActiveSpace];
	[panel setContentView:all];
	[panel setContentSize:[all fittingSize]];
	access.panel = panel;
	return access;
}

// csAskAccess shows why Accessibility is needed, with a button that opens
// its settings. It returns at once and reuses one panel.
void csAskAccess(const char *window, const char *title, const char *body, const char *open, const char *later) {
	@autoreleasepool {
		if (csAccess == nil) {
			csAccess = csBuildAccessPanel(window, title, body, open, later);
		}
		csActivateSelf();
		[csAccess.panel center];
		[csAccess.panel makeKeyAndOrderFront:nil];
	}
}

// csCloseAccess hides the explanation once it is no longer needed.
void csCloseAccess(void) {
	[csAccess.panel orderOut:nil];
}

// csToast shows text in a small panel under the menu bar for a few seconds.
// The panel never takes focus or clicks.
void csToast(const char *text) {
	@autoreleasepool {
		NSTextField *label = [NSTextField labelWithString:@(text)];
		[label setFont:[NSFont systemFontOfSize:13 weight:NSFontWeightMedium]];
		[label sizeToFit];
		NSSize size = [label frame].size;
		NSRect frame = NSMakeRect(0, 0, size.width + 32, size.height + 20);
		[label setFrameOrigin:NSMakePoint(16, 10)];

		NSVisualEffectView *back = [[[NSVisualEffectView alloc] initWithFrame:frame] autorelease];
		[back setMaterial:NSVisualEffectMaterialHUDWindow];
		[back setState:NSVisualEffectStateActive];
		[back setWantsLayer:YES];
		[[back layer] setCornerRadius:10];
		[[back layer] setMasksToBounds:YES];
		[back addSubview:label];

		NSPanel *panel = [[NSPanel alloc] initWithContentRect:frame
		                                            styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
		                                              backing:NSBackingStoreBuffered
		                                                defer:NO];
		[panel setReleasedWhenClosed:NO];
		[panel setOpaque:NO];
		[panel setBackgroundColor:[NSColor clearColor]];
		[panel setHasShadow:YES];
		[panel setIgnoresMouseEvents:YES];
		[panel setLevel:NSStatusWindowLevel];
		[panel setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces |
		                             NSWindowCollectionBehaviorTransient |
		                             NSWindowCollectionBehaviorIgnoresCycle];
		[panel setContentView:back];

		NSArray *screens = [NSScreen screens];
		if ([screens count] > 0) {
			NSRect visible = [[screens objectAtIndex:0] visibleFrame];
			[panel setFrameOrigin:NSMakePoint(NSMaxX(visible) - frame.size.width - 16,
			                                  NSMaxY(visible) - frame.size.height - 12)];
		}
		[panel orderFrontRegardless];
		dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 4 * NSEC_PER_SEC), dispatch_get_main_queue(), ^{
			[panel orderOut:nil];
			[panel release];
		});
	}
}
