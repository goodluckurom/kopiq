//go:build darwin

package icons

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#import <stdlib.h>

static unsigned char* iconPNGForBundleID(const char* bundleID, int* outLen) {
    *outLen = 0;
    if (bundleID == NULL || bundleID[0] == '\0') return NULL;

    @autoreleasepool {
        NSString *bid = [NSString stringWithUTF8String:bundleID];
        NSURL *url = [[NSWorkspace sharedWorkspace] URLForApplicationWithBundleIdentifier:bid];
        NSImage *icon = nil;
        if (url != nil) {
            icon = [[NSWorkspace sharedWorkspace] iconForFile:url.path];
        }
        if (icon == nil) {
            NSRunningApplication *front = [[NSWorkspace sharedWorkspace] frontmostApplication];
            if (front != nil && [front.bundleIdentifier isEqualToString:bid]) {
                icon = front.icon;
            }
        }
        if (icon == nil) return NULL;

        NSSize size = NSMakeSize(128, 128);
        NSImage *scaled = [[NSImage alloc] initWithSize:size];
        [scaled lockFocus];
        [[NSGraphicsContext currentContext] setImageInterpolation:NSImageInterpolationHigh];
        [icon drawInRect:NSMakeRect(0, 0, size.width, size.height)
                fromRect:NSZeroRect
               operation:NSCompositingOperationCopy
                fraction:1.0];
        [scaled unlockFocus];

        NSData *tiff = [scaled TIFFRepresentation];
        if (tiff == nil) return NULL;
        NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiff];
        if (rep == nil) return NULL;
        NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        if (png == nil || png.length == 0) return NULL;

        unsigned char *buf = (unsigned char *)malloc(png.length);
        if (buf == NULL) return NULL;
        memcpy(buf, png.bytes, png.length);
        *outLen = (int)png.length;
        return buf;
    }
}

static char* displayNameForBundleID(const char* bundleID) {
    if (bundleID == NULL || bundleID[0] == '\0') return NULL;
    @autoreleasepool {
        NSString *bid = [NSString stringWithUTF8String:bundleID];
        NSURL *url = [[NSWorkspace sharedWorkspace] URLForApplicationWithBundleIdentifier:bid];
        if (url == nil) return NULL;

        NSString *name = nil;
        NSBundle *bundle = [NSBundle bundleWithURL:url];
        if (bundle != nil) {
            name = [bundle objectForInfoDictionaryKey:@"CFBundleDisplayName"];
            if (name == nil || name.length == 0) {
                name = [bundle objectForInfoDictionaryKey:@"CFBundleName"];
            }
        }
        if (name == nil || name.length == 0) {
            name = [[NSFileManager defaultManager] displayNameAtPath:url.path];
        }
        if (name == nil || name.length == 0) return NULL;
        return strdup([name UTF8String]);
    }
}

static int resolveAppByName(const char* name, char** outBundleID, char** outDisplayName) {
    *outBundleID = NULL;
    *outDisplayName = NULL;
    if (name == NULL || name[0] == '\0') return 0;

    @autoreleasepool {
        NSString *query = [[NSString stringWithUTF8String:name] stringByTrimmingCharactersInSet:
            [NSCharacterSet whitespaceAndNewlineCharacterSet]];
        if (query.length == 0) return 0;

        NSString *path = nil;
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
        path = [[NSWorkspace sharedWorkspace] fullPathForApplication:query];
#pragma clang diagnostic pop

        NSRunningApplication *matched = nil;
        if (path == nil) {
            NSArray<NSRunningApplication *> *apps = [[NSWorkspace sharedWorkspace] runningApplications];
            for (NSRunningApplication *app in apps) {
                NSString *loc = app.localizedName;
                if (loc == nil) continue;
                if ([loc caseInsensitiveCompare:query] == NSOrderedSame) {
                    matched = app;
                    break;
                }
            }
            if (matched == nil) {
                NSString *lower = [query lowercaseString];
                for (NSRunningApplication *app in apps) {
                    NSString *loc = app.localizedName;
                    if (loc == nil) continue;
                    if ([[loc lowercaseString] containsString:lower]) {
                        matched = app;
                        break;
                    }
                }
            }
            if (matched != nil && matched.bundleURL != nil) {
                path = matched.bundleURL.path;
            }
        }

        if (path == nil || path.length == 0) return 0;

        NSBundle *bundle = [NSBundle bundleWithPath:path];
        NSString *bid = bundle.bundleIdentifier;
        if ((bid == nil || bid.length == 0) && matched != nil) {
            bid = matched.bundleIdentifier;
        }
        if (bid == nil || bid.length == 0) return 0;

        NSString *display = nil;
        if (matched != nil && matched.localizedName.length > 0) {
            display = matched.localizedName;
        }
        if (display == nil || display.length == 0) {
            display = [bundle objectForInfoDictionaryKey:@"CFBundleDisplayName"];
        }
        if (display == nil || display.length == 0) {
            display = [bundle objectForInfoDictionaryKey:@"CFBundleName"];
        }
        if (display == nil || display.length == 0) {
            display = [[NSFileManager defaultManager] displayNameAtPath:path];
        }
        if (display == nil || display.length == 0) {
            display = query;
        }

        *outBundleID = strdup([bid UTF8String]);
        *outDisplayName = strdup([display UTF8String]);
        return 1;
    }
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

func extractPNG(bundleID string) ([]byte, error) {
	if bundleID == "" {
		return nil, nil
	}
	cBid := C.CString(bundleID)
	defer C.free(unsafe.Pointer(cBid))

	var n C.int
	ptr := C.iconPNGForBundleID(cBid, &n)
	if ptr == nil || n <= 0 {
		return nil, fmt.Errorf("no icon for %s", bundleID)
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), n), nil
}

func platformDisplayName(appID string) string {
	if appID == "" {
		return ""
	}
	cBid := C.CString(appID)
	defer C.free(unsafe.Pointer(cBid))
	ptr := C.displayNameForBundleID(cBid)
	if ptr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoString(ptr)
}

func platformResolveByName(name string) (bundleID, displayName string, ok bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", false
	}
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	var cBid, cDisp *C.char
	if C.resolveAppByName(cName, &cBid, &cDisp) == 0 {
		return "", "", false
	}
	if cBid != nil {
		defer C.free(unsafe.Pointer(cBid))
		bundleID = C.GoString(cBid)
	}
	if cDisp != nil {
		defer C.free(unsafe.Pointer(cDisp))
		displayName = C.GoString(cDisp)
	}
	return bundleID, displayName, bundleID != ""
}
