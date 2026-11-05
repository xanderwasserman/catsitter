#import <AppKit/AppKit.h>
#import <ApplicationServices/ApplicationServices.h>

static void csActivateSelf(void) {
	if (@available(macOS 14, *)) {
		[NSApp activate];
	} else {
		[NSApp activateIgnoringOtherApps:YES];
	}
}

// csOpenAccessSettings adds Catsitter to the Accessibility list, then opens
// Privacy & Security > Accessibility. Asking with the prompt option is the
// documented way to get listed. macOS may show its own dialog for it too.
void csOpenAccessSettings(void) {
	@autoreleasepool {
		NSDictionary *ask = @{(id)kAXTrustedCheckOptionPrompt: @YES};
		AXIsProcessTrustedWithOptions((CFDictionaryRef)ask);
		NSURL *url = [NSURL URLWithString:@"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"];
		[[NSWorkspace sharedWorkspace] openURL:url];
	}
}

@interface CSAccessPanel : NSObject
@property (nonatomic, retain) NSPanel *panel;
@end

@implementation CSAccessPanel
- (void)openSettings:(id)sender {
	csOpenAccessSettings();
	[self.panel orderOut:nil];
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
