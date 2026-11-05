// click X Y [right] posts one mouse click for packaging/test-app.sh.
#import <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>
#include <unistd.h>

static void post(CGEventType type, CGPoint p, CGMouseButton button) {
	CGEventRef e = CGEventCreateMouseEvent(NULL, type, p, button);
	CGEventPost(kCGHIDEventTap, e);
	CFRelease(e);
	usleep(80000);
}

int main(int argc, char **argv) {
	if (argc < 3) {
		return 2;
	}
	CGPoint p = CGPointMake(atof(argv[1]), atof(argv[2]));
	int right = argc > 3;
	CGMouseButton button = right ? kCGMouseButtonRight : kCGMouseButtonLeft;
	post(kCGEventMouseMoved, p, button);
	post(right ? kCGEventRightMouseDown : kCGEventLeftMouseDown, p, button);
	post(right ? kCGEventRightMouseUp : kCGEventLeftMouseUp, p, button);
	return 0;
}
