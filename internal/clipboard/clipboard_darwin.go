//go:build darwin

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#include <unistd.h>

static long pbChangeCount(void) {
    return (long)[[NSPasteboard generalPasteboard] changeCount];
}

static char* pbReadString(void) {
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        NSString *s = [pb stringForType:NSPasteboardTypeString];
        if (s == nil) {
            return NULL;
        }
        return strdup([s UTF8String]);
    }
}

static long pbWriteString(const char* text) {
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        [pb clearContents];
        NSString *s = [NSString stringWithUTF8String:text];
        [pb setString:s forType:NSPasteboardTypeString];
        return (long)[pb changeCount];
    }
}

static int pbHasImage(void) {
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        NSArray *types = [pb types];
        if ([types containsObject:NSPasteboardTypePNG] ||
            [types containsObject:NSPasteboardTypeTIFF] ||
            [types containsObject:@"public.jpeg"] ||
            [types containsObject:@"public.heic"]) {
            return 1;
        }
        // NSImage can often synthesize from other representations.
        NSImage *img = [[NSImage alloc] initWithPasteboard:pb];
        return (img != nil && img.isValid) ? 1 : 0;
    }
}

// Prefer PNG, then TIFF, then JPEG, then GIF/WebP, then NSImage→PNG.
// Caller frees *outType and *outData.
static int pbReadImageOnce(unsigned char **outData, int *outLen, char **outType) {
    *outData = NULL;
    *outLen = 0;
    *outType = NULL;
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        NSData *data = [pb dataForType:NSPasteboardTypePNG];
        const char *type = "png";
        if (data == nil || data.length == 0) {
            data = [pb dataForType:NSPasteboardTypeTIFF];
            type = "tiff";
        }
        if (data == nil || data.length == 0) {
            data = [pb dataForType:@"public.jpeg"];
            type = "jpeg";
        }
        if (data == nil || data.length == 0) {
            data = [pb dataForType:@"public.gif"];
            type = "gif";
        }
        if (data == nil || data.length == 0) {
            data = [pb dataForType:@"public.webp"];
            type = "webp";
        }
        if (data == nil || data.length == 0) {
            // Last resort: let AppKit coerce whatever is on the pasteboard into PNG.
            NSImage *img = [[NSImage alloc] initWithPasteboard:pb];
            if (img != nil && img.isValid) {
                NSData *tiff = [img TIFFRepresentation];
                if (tiff != nil) {
                    NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiff];
                    if (rep != nil) {
                        data = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
                        type = "png";
                    }
                }
            }
        }
        if (data == nil || data.length == 0) {
            return 0;
        }
        unsigned char *buf = (unsigned char *)malloc(data.length);
        if (buf == NULL) return 0;
        memcpy(buf, data.bytes, data.length);
        *outData = buf;
        *outLen = (int)data.length;
        *outType = strdup(type);
        return 1;
    }
}

static int pbPasteboardLooksLikeImage(void) {
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        NSArray *types = [pb types];
        NSArray *want = @[
            NSPasteboardTypePNG,
            NSPasteboardTypeTIFF,
            @"public.jpeg",
            @"public.jpg",
            @"public.heic",
            @"public.gif",
            @"public.webp",
            @"com.compuserve.gif",
            @"com.microsoft.bmp",
            NSPasteboardTypeFileURL
        ];
        for (NSString *t in want) {
            if ([types containsObject:t]) return 1;
        }
        NSImage *img = [[NSImage alloc] initWithPasteboard:pb];
        return (img != nil && img.isValid) ? 1 : 0;
    }
}

// Prefer PNG, then TIFF, then JPEG, then NSImage→PNG. Caller frees *outType and *outData.
// Retries briefly when the pasteboard advertises an image but promised data is not ready yet
// (common for browser "Copy Image" and Finder quick Look transfers).
static int pbReadImage(unsigned char **outData, int *outLen, char **outType) {
    if (pbReadImageOnce(outData, outLen, outType) == 1) {
        return 1;
    }
    if (!pbPasteboardLooksLikeImage()) {
        return 0;
    }
    for (int i = 0; i < 6; i++) {
        usleep(40000); // 40ms × 6 ≈ 240ms
        if (pbReadImageOnce(outData, outLen, outType) == 1) {
            return 1;
        }
    }
    return 0;
}

// Collect file:// paths from the pasteboard. Returns count; *outPaths is a
// malloc'd array of malloc'd C strings the caller must free.
static int pbReadFilePaths(char ***outPaths) {
    *outPaths = NULL;
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        NSMutableArray<NSString *> *paths = [NSMutableArray array];

        NSArray *urls = [pb readObjectsForClasses:@[[NSURL class]]
                                          options:@{ NSPasteboardURLReadingFileURLsOnlyKey: @YES }];
        for (NSURL *u in urls) {
            if (u.isFileURL && u.path.length > 0) {
                [paths addObject:u.path];
            }
        }

        // Legacy filenames property list.
        if (paths.count == 0) {
            NSArray *names = [pb propertyListForType:NSPasteboardTypeFileURL];
            // Also try NSFilenamesPboardType via string constant.
            if (![names isKindOfClass:[NSArray class]]) {
                names = [pb propertyListForType:@"NSFilenamesPboardType"];
            }
            if ([names isKindOfClass:[NSArray class]]) {
                for (id n in names) {
                    if ([n isKindOfClass:[NSString class]] && [(NSString *)n length] > 0) {
                        [paths addObject:(NSString *)n];
                    } else if ([n isKindOfClass:[NSURL class]]) {
                        NSURL *u = (NSURL *)n;
                        if (u.isFileURL && u.path.length > 0) [paths addObject:u.path];
                    }
                }
            }
        }

        if (paths.count == 0) return 0;

        char **arr = (char **)calloc(paths.count, sizeof(char *));
        if (arr == NULL) return 0;
        for (NSUInteger i = 0; i < paths.count; i++) {
            arr[i] = strdup([paths[i] UTF8String]);
        }
        *outPaths = arr;
        return (int)paths.count;
    }
}

static long pbWritePNG(const unsigned char *data, int len) {
    @autoreleasepool {
        if (data == NULL || len <= 0) return 0;
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        [pb clearContents];
        NSData *ns = [NSData dataWithBytes:data length:(NSUInteger)len];
        [pb setData:ns forType:NSPasteboardTypePNG];
        return (long)[pb changeCount];
    }
}

static void freeCString(char* s) {
    if (s) free(s);
}

static void freeCStringArray(char **arr, int n) {
    if (arr == NULL) return;
    for (int i = 0; i < n; i++) {
        if (arr[i]) free(arr[i]);
    }
    free(arr);
}
*/
import "C"

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"kopiq/internal/domain"
)

type darwinSource struct {
	mu       sync.Mutex
	ch       chan domain.Generation
	cancel   context.CancelFunc
	baseline domain.Generation
	running  bool
}

// New returns the platform clipboard source.
func New() Source {
	return &darwinSource{
		ch: make(chan domain.Generation, 16),
	}
}

func (d *darwinSource) SupportsBackgroundWatch() bool { return true }

func (d *darwinSource) Changes() <-chan domain.Generation { return d.ch }

func (d *darwinSource) CurrentGeneration() (domain.Generation, error) {
	return domain.Generation(C.pbChangeCount()), nil
}

func (d *darwinSource) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		return nil
	}
	gen, err := d.CurrentGeneration()
	if err != nil {
		return err
	}
	d.baseline = gen
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.running = true
	go d.poll(ctx)
	return nil
}

func (d *darwinSource) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.running {
		return nil
	}
	d.cancel()
	d.running = false
	return nil
}

func (d *darwinSource) poll(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	last := d.baseline
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cur := domain.Generation(C.pbChangeCount())
			if cur != last {
				last = cur
				select {
				case d.ch <- cur:
				default:
					// drop if pipeline is backed up; next change will catch up
				}
			}
		}
	}
}

func (d *darwinSource) Read(gen domain.Generation) (domain.RawPayload, error) {
	cur := domain.Generation(C.pbChangeCount())
	_ = gen
	payload := domain.RawPayload{
		Generation: cur,
		OSTypes:    []string{},
	}
	cstr := C.pbReadString()
	if cstr != nil {
		payload.Text = C.GoString(cstr)
		C.freeCString(cstr)
		payload.OSTypes = append(payload.OSTypes, "public.utf8-plain-text")
	}

	var paths **C.char
	n := int(C.pbReadFilePaths(&paths))
	if n > 0 && paths != nil {
		slice := unsafe.Slice(paths, n)
		for i := 0; i < n; i++ {
			if slice[i] != nil {
				payload.FileURLs = append(payload.FileURLs, C.GoString(slice[i]))
			}
		}
		C.freeCStringArray(paths, C.int(n))
		payload.OSTypes = append(payload.OSTypes, "public.file-url")
	}

	var imgData *C.uchar
	var imgLen C.int
	var imgType *C.char
	if C.pbReadImage(&imgData, &imgLen, &imgType) == 1 && imgData != nil && imgLen > 0 {
		payload.ImageBytes = C.GoBytes(unsafe.Pointer(imgData), imgLen)
		C.free(unsafe.Pointer(imgData))
		if imgType != nil {
			payload.ImageType = C.GoString(imgType)
			C.freeCString(imgType)
		}
		switch payload.ImageType {
		case "tiff":
			payload.OSTypes = append(payload.OSTypes, "public.tiff")
		case "jpeg":
			payload.OSTypes = append(payload.OSTypes, "public.jpeg")
		default:
			payload.OSTypes = append(payload.OSTypes, "public.png")
		}
	}
	return payload, nil
}

func (d *darwinSource) Write(text string) (domain.Generation, error) {
	cstr := C.CString(text)
	defer C.free(unsafe.Pointer(cstr))
	gen := domain.Generation(C.pbWriteString(cstr))
	if gen == 0 {
		return 0, fmt.Errorf("clipboard write failed")
	}
	return gen, nil
}

// WriteImage places PNG bytes on the pasteboard and returns the new generation.
func (d *darwinSource) WriteImage(png []byte) (domain.Generation, error) {
	if len(png) == 0 {
		return 0, fmt.Errorf("empty png")
	}
	gen := domain.Generation(C.pbWritePNG((*C.uchar)(unsafe.Pointer(&png[0])), C.int(len(png))))
	if gen == 0 {
		return 0, fmt.Errorf("clipboard image write failed")
	}
	return gen, nil
}
