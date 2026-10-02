//go:build darwin

package paste

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework ApplicationServices -framework Carbon
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <stdlib.h>
#import <string.h>

static char* frontAppName(void) {
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (app == nil || app.localizedName == nil) return NULL;
        return strdup([app.localizedName UTF8String]);
    }
}

static char* frontAppBundleID(void) {
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (app == nil || app.bundleIdentifier == nil) return NULL;
        return strdup([app.bundleIdentifier UTF8String]);
    }
}

static int frontAppPID(void) {
    @autoreleasepool {
        NSRunningApplication *app = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (app == nil) return 0;
        return (int)app.processIdentifier;
    }
}

static void freeCString(char* s) {
    if (s) free(s);
}

static bool axTrusted(void) {
    NSDictionary *opts = @{(__bridge id)kAXTrustedCheckOptionPrompt: @NO};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts);
}

static bool axPrompt(void) {
    NSDictionary *opts = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts);
}

// Activate by pid. Returns 1 on success.
static int activatePID(int pid) {
    @autoreleasepool {
        if (pid <= 0) return 0;
        NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:(pid_t)pid];
        if (app == nil) return 0;
        BOOL ok = [app activateWithOptions:NSApplicationActivateIgnoringOtherApps];
        return ok ? 1 : 0;
    }
}

// Activate by bundle id. Returns 1 on success.
static int activateBundle(const char *bundleID) {
    @autoreleasepool {
        if (bundleID == NULL) return 0;
        NSString *bid = [NSString stringWithUTF8String:bundleID];
        NSArray<NSRunningApplication *> *apps =
            [NSRunningApplication runningApplicationsWithBundleIdentifier:bid];
        if (apps.count == 0) return 0;
        BOOL ok = [apps.firstObject activateWithOptions:NSApplicationActivateIgnoringOtherApps];
        return ok ? 1 : 0;
    }
}

// Post Cmd+V. Requires Accessibility on modern macOS for posting to other apps.
static int sendPasteKey(void) {
    if (!axTrusted()) return 0;
    CGEventSourceRef src = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
    if (src == NULL) return 0;

    CGEventRef down = CGEventCreateKeyboardEvent(src, (CGKeyCode)kVK_ANSI_V, true);
    CGEventRef up = CGEventCreateKeyboardEvent(src, (CGKeyCode)kVK_ANSI_V, false);
    if (down == NULL || up == NULL) {
        if (down) CFRelease(down);
        if (up) CFRelease(up);
        CFRelease(src);
        return 0;
    }
    CGEventSetFlags(down, kCGEventFlagMaskCommand);
    CGEventSetFlags(up, kCGEventFlagMaskCommand);
    CGEventPost(kCGHIDEventTap, down);
    CGEventPost(kCGHIDEventTap, up);
    CFRelease(down);
    CFRelease(up);
    CFRelease(src);
    return 1;
}
*/
import "C"

import (
	"fmt"
	"os/exec"
	"time"
	"unsafe"
)

// CaptureTarget snapshots the frontmost app before Quick Paste is shown.
func CaptureTarget() (Target, error) {
	t := Target{
		ProcessID: int(C.frontAppPID()),
	}
	if name := C.frontAppName(); name != nil {
		t.ApplicationName = C.GoString(name)
		C.freeCString(name)
	}
	if bid := C.frontAppBundleID(); bid != nil {
		t.ApplicationID = C.GoString(bid)
		C.freeCString(bid)
	}
	if !t.Valid() {
		return t, fmt.Errorf("no frontmost application")
	}
	return t, nil
}

// CanSynthesizePaste reports whether Accessibility is granted for key posting.
func CanSynthesizePaste() bool {
	return bool(C.axTrusted())
}

// PromptAccessibility shows the system Accessibility trust dialog when needed.
func PromptAccessibility() bool {
	return bool(C.axPrompt())
}

// OpenAccessibilitySettings opens macOS Privacy → Accessibility.
func OpenAccessibilitySettings() error {
	urls := []string{
		"x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?Privacy_Accessibility",
		"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility",
	}
	var last error
	for _, u := range urls {
		if err := exec.Command("open", u).Start(); err == nil {
			return nil
		} else {
			last = err
		}
	}
	return last
}

// Activate brings the target app forward.
func Activate(t Target) error {
	if t.ProcessID > 0 && C.activatePID(C.int(t.ProcessID)) == 1 {
		return nil
	}
	if t.ApplicationID != "" {
		cstr := C.CString(t.ApplicationID)
		defer C.free(unsafe.Pointer(cstr))
		if C.activateBundle(cstr) == 1 {
			return nil
		}
	}
	return fmt.Errorf("could not activate %s", t.String())
}

// SendPaste posts Command+V.
func SendPaste() error {
	if C.sendPasteKey() != 1 {
		return fmt.Errorf("could not synthesize paste (Accessibility required)")
	}
	return nil
}

// Settle briefly after activation so the target accepts the paste.
func Settle() {
	time.Sleep(40 * time.Millisecond)
}
