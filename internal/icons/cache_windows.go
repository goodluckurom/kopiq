//go:build windows

package icons

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modShell32                 = windows.NewLazySystemDLL("shell32.dll")
	modUser32                  = windows.NewLazySystemDLL("user32.dll")
	modGdi32                   = windows.NewLazySystemDLL("gdi32.dll")
	procSHGetFileInfoW         = modShell32.NewProc("SHGetFileInfoW")
	procDestroyIcon            = modUser32.NewProc("DestroyIcon")
	procGetDC                  = modUser32.NewProc("GetDC")
	procReleaseDC              = modUser32.NewProc("ReleaseDC")
	procCreateCompatibleDC     = modGdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = modGdi32.NewProc("SelectObject")
	procDeleteObject           = modGdi32.NewProc("DeleteObject")
	procDeleteDC               = modGdi32.NewProc("DeleteDC")
	procDrawIconEx             = modUser32.NewProc("DrawIconEx")
	procGetDIBits              = modGdi32.NewProc("GetDIBits")
)

const (
	shgfiIcon      = 0x000000100
	shgfiLargeIcon = 0x000000000
	diNormal       = 0x0003
)

type shFileInfo struct {
	HIcon         windows.Handle
	IIcon         int32
	DwAttributes  uint32
	SzDisplayName [260]uint16
	SzTypeName    [80]uint16
}

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// extractPNG on Windows treats appID as an executable path (preferred) or leaves empty.
func extractPNG(appID string) ([]byte, error) {
	if appID == "" {
		return nil, nil
	}
	path, err := windows.UTF16PtrFromString(appID)
	if err != nil {
		return nil, err
	}
	var sfi shFileInfo
	ret, _, callErr := procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(path)),
		0,
		uintptr(unsafe.Pointer(&sfi)),
		uintptr(unsafe.Sizeof(sfi)),
		uintptr(shgfiIcon|shgfiLargeIcon),
	)
	if ret == 0 || sfi.HIcon == 0 {
		if callErr != syscall.Errno(0) {
			return nil, callErr
		}
		return nil, fmt.Errorf("no icon for %s", appID)
	}
	defer procDestroyIcon.Call(uintptr(sfi.HIcon))

	img, err := hiconToRGBA(sfi.HIcon, 128)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hiconToRGBA(hIcon windows.Handle, size int) (*image.RGBA, error) {
	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(hdcMem)

	hbm, _, _ := procCreateCompatibleBitmap.Call(hdcScreen, uintptr(size), uintptr(size))
	if hbm == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer procDeleteObject.Call(hbm)

	old, _, _ := procSelectObject.Call(hdcMem, hbm)
	defer procSelectObject.Call(hdcMem, old)

	// Clear to transparent-ish black then draw icon.
	procDrawIconEx.Call(
		hdcMem,
		0, 0,
		uintptr(hIcon),
		uintptr(size), uintptr(size),
		0, 0,
		uintptr(diNormal),
	)

	bi := bitmapInfoHeader{
		BiSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		BiWidth:       int32(size),
		BiHeight:      -int32(size), // top-down
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0, // BI_RGB
	}
	buf := make([]byte, size*size*4)
	ret, _, _ := procGetDIBits.Call(
		hdcMem,
		hbm,
		0,
		uintptr(size),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bi)),
		0, // DIB_RGB_COLORS
	)
	if ret == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			i := (y*size + x) * 4
			// Windows BGRA → RGBA
			b, g, r, a := buf[i+0], buf[i+1], buf[i+2], buf[i+3]
			off := img.PixOffset(x, y)
			img.Pix[off+0] = r
			img.Pix[off+1] = g
			img.Pix[off+2] = b
			img.Pix[off+3] = a
		}
	}
	return img, nil
}
