//go:build darwin

package dock

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>

static void setDockVisible(int show) {
    dispatch_block_t block = ^{
        if (show) {
            [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        } else {
            [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        }
    };
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_sync(dispatch_get_main_queue(), block);
    }
}
*/
import "C"

func setVisible(show bool) {
	v := C.int(0)
	if show {
		v = 1
	}
	C.setDockVisible(v)
}
