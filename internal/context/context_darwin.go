//go:build darwin

package context

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework ApplicationServices
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#import <ApplicationServices/ApplicationServices.h>
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
    // Don't prompt here during capture — settings/onboarding owns the prompt.
    NSDictionary *opts = @{(__bridge id)kAXTrustedCheckOptionPrompt: @NO};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts);
}

static char* copyAXString(AXUIElementRef el, CFStringRef attr) {
    if (el == NULL) return NULL;
    CFTypeRef value = NULL;
    if (AXUIElementCopyAttributeValue(el, attr, &value) != kAXErrorSuccess || value == NULL) {
        return NULL;
    }
    char *out = NULL;
    if (CFGetTypeID(value) == CFStringGetTypeID()) {
        NSString *s = (__bridge NSString *)value;
        if (s.length > 0) {
            out = strdup([s UTF8String]);
        }
    }
    CFRelease(value);
    return out;
}

// Best-effort deep fields for the focused window of pid.
// Out params are heap strings the caller must freeCString, or NULL.
static void captureDeepAX(int pid, char **windowTitle, char **documentPath, char **url) {
    *windowTitle = NULL;
    *documentPath = NULL;
    *url = NULL;
    if (pid <= 0 || !axTrusted()) return;

    AXUIElementRef app = AXUIElementCreateApplication((pid_t)pid);
    if (app == NULL) return;

    // Keep AX calls snappy so we never stall the capture pipeline.
    AXUIElementSetMessagingTimeout(app, 0.04);

    AXUIElementRef window = NULL;
    if (AXUIElementCopyAttributeValue(app, kAXFocusedWindowAttribute, (CFTypeRef *)&window) != kAXErrorSuccess || window == NULL) {
        CFRelease(app);
        return;
    }

    *windowTitle = copyAXString(window, kAXTitleAttribute);

    // Document path when the app exposes AXDocument (editors, Preview, etc.).
    CFTypeRef doc = NULL;
    if (AXUIElementCopyAttributeValue(window, kAXDocumentAttribute, &doc) == kAXErrorSuccess && doc != NULL) {
        if (CFGetTypeID(doc) == CFStringGetTypeID()) {
            NSString *s = (__bridge NSString *)doc;
            // Often a file:// URL
            if ([s hasPrefix:@"file://"]) {
                NSURL *u = [NSURL URLWithString:s];
                if (u.path.length > 0) {
                    *documentPath = strdup([u.path UTF8String]);
                }
            } else if (s.length > 0) {
                *documentPath = strdup([s UTF8String]);
            }
        } else if (CFGetTypeID(doc) == CFURLGetTypeID()) {
            NSURL *u = (__bridge NSURL *)doc;
            if (u.path.length > 0) {
                *documentPath = strdup([u.path UTF8String]);
            }
        }
        CFRelease(doc);
    }

    // Browser URL: walk a few children for AXURL / AXValue that look like http(s).
    // Misses are fine — we keep window title.
    CFArrayRef children = NULL;
    if (AXUIElementCopyAttributeValue(window, kAXChildrenAttribute, (CFTypeRef *)&children) == kAXErrorSuccess && children != NULL) {
        CFIndex count = CFArrayGetCount(children);
        if (count > 12) count = 12;
        for (CFIndex i = 0; i < count && *url == NULL; i++) {
            AXUIElementRef child = (AXUIElementRef)CFArrayGetValueAtIndex(children, i);
            if (child == NULL) continue;

            CFTypeRef urlVal = NULL;
            if (AXUIElementCopyAttributeValue(child, kAXURLAttribute, &urlVal) == kAXErrorSuccess && urlVal != NULL) {
                if (CFGetTypeID(urlVal) == CFURLGetTypeID()) {
                    NSURL *u = (__bridge NSURL *)urlVal;
                    NSString *abs = u.absoluteString;
                    if ([abs hasPrefix:@"http://"] || [abs hasPrefix:@"https://"]) {
                        *url = strdup([abs UTF8String]);
                    }
                } else if (CFGetTypeID(urlVal) == CFStringGetTypeID()) {
                    NSString *s = (__bridge NSString *)urlVal;
                    if ([s hasPrefix:@"http://"] || [s hasPrefix:@"https://"]) {
                        *url = strdup([s UTF8String]);
                    }
                }
                CFRelease(urlVal);
            }

            if (*url == NULL) {
                char *val = copyAXString(child, kAXValueAttribute);
                if (val != NULL) {
                    if (strncmp(val, "http://", 7) == 0 || strncmp(val, "https://", 8) == 0) {
                        *url = val;
                    } else {
                        free(val);
                    }
                }
            }
        }
        CFRelease(children);
    }

    CFRelease(window);
    CFRelease(app);
}
*/
import "C"

import (
	"path/filepath"
	"strings"
	"time"

	"kopiq/internal/domain"
	"kopiq/internal/icons"
	"kopiq/internal/redact"
)

type darwinProvider struct{}

// New returns the platform context provider.
func New() Provider {
	return &darwinProvider{}
}

func (d *darwinProvider) Capture(deep bool) (domain.Snapshot, error) {
	snap := domain.Snapshot{
		ApplicationName: "Unknown",
		CapturedAt:      time.Now().UnixMilli(),
		Deep:            false,
	}
	if name := C.frontAppName(); name != nil {
		snap.ApplicationName = C.GoString(name)
		C.freeCString(name)
	}
	if bid := C.frontAppBundleID(); bid != nil {
		snap.ApplicationID = C.GoString(bid)
		C.freeCString(bid)
	}
	snap.ProcessID = int(C.frontAppPID())
	if snap.ApplicationID != "" {
		if key, err := icons.Ensure(snap.ApplicationID); err == nil && key != "" {
			snap.IconCacheKey = key
		} else {
			snap.IconCacheKey = icons.KeyFromAppID(snap.ApplicationID)
		}
	}

	if !deep {
		return snap, nil
	}

	var windowTitle, documentPath, pageURL *C.char
	C.captureDeepAX(C.int(snap.ProcessID), &windowTitle, &documentPath, &pageURL)
	if windowTitle != nil {
		snap.WindowTitle = C.GoString(windowTitle)
		C.freeCString(windowTitle)
		snap.Deep = true
	}
	if documentPath != nil {
		path := C.GoString(documentPath)
		C.freeCString(documentPath)
		// Chrome/Safari sometimes expose the page URL as AXDocument — that belongs in URL, not File.
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			if pageURL == nil {
				if clean, domain := redact.URL(path); clean != "" {
					snap.URL = clean
					snap.Domain = domain
					if snap.DocumentTitle == "" && snap.WindowTitle != "" {
						snap.DocumentTitle = snap.WindowTitle
					}
					snap.Deep = true
				}
			}
		} else {
			snap.FilePath = path
			snap.FileName = filepath.Base(path)
			snap.Deep = true
		}
	}
	if pageURL != nil {
		raw := C.GoString(pageURL)
		C.freeCString(pageURL)
		if clean, domain := redact.URL(raw); clean != "" {
			snap.URL = clean
			snap.Domain = domain
			if snap.DocumentTitle == "" && snap.WindowTitle != "" {
				snap.DocumentTitle = snap.WindowTitle
			}
			snap.Deep = true
		}
	}

	return snap, nil
}
