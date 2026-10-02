//go:build windows

package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"kopiq/internal/domain"
)

var (
	modUser32                      = windows.NewLazySystemDLL("user32.dll")
	modKernel32                    = windows.NewLazySystemDLL("kernel32.dll")
	procGetClipboardSequenceNumber = modUser32.NewProc("GetClipboardSequenceNumber")
	procOpenClipboard              = modUser32.NewProc("OpenClipboard")
	procCloseClipboard             = modUser32.NewProc("CloseClipboard")
	procEmptyClipboard             = modUser32.NewProc("EmptyClipboard")
	procGetClipboardData           = modUser32.NewProc("GetClipboardData")
	procSetClipboardData           = modUser32.NewProc("SetClipboardData")
	procIsClipboardFormatAvailable = modUser32.NewProc("IsClipboardFormatAvailable")
	procRegisterClipboardFormatW   = modUser32.NewProc("RegisterClipboardFormatW")
	procGlobalAlloc                = modKernel32.NewProc("GlobalAlloc")
	procGlobalLock                 = modKernel32.NewProc("GlobalLock")
	procGlobalUnlock               = modKernel32.NewProc("GlobalUnlock")
	procGlobalSize                 = modKernel32.NewProc("GlobalSize")
)

const (
	cfUnicodeText = 13
	cfDIB         = 8
	gmemMoveable  = 0x0002
)

var cfPNG uint32

func init() {
	name, _ := windows.UTF16PtrFromString("PNG")
	r, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name)))
	cfPNG = uint32(r)
}

type windowsSource struct {
	mu       sync.Mutex
	ch       chan domain.Generation
	cancel   context.CancelFunc
	baseline domain.Generation
	running  bool
}

// New returns the platform clipboard source.
func New() Source {
	return &windowsSource{
		ch: make(chan domain.Generation, 16),
	}
}

func (w *windowsSource) SupportsBackgroundWatch() bool { return true }

func (w *windowsSource) Changes() <-chan domain.Generation { return w.ch }

func (w *windowsSource) CurrentGeneration() (domain.Generation, error) {
	n, _, _ := procGetClipboardSequenceNumber.Call()
	return domain.Generation(n), nil
}

func (w *windowsSource) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return nil
	}
	gen, err := w.CurrentGeneration()
	if err != nil {
		return err
	}
	w.baseline = gen
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.running = true
	go w.poll(ctx)
	return nil
}

func (w *windowsSource) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.running {
		return nil
	}
	w.cancel()
	w.running = false
	return nil
}

func (w *windowsSource) poll(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	last := w.baseline
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gen, err := w.CurrentGeneration()
			if err != nil || gen == last {
				continue
			}
			last = gen
			select {
			case w.ch <- gen:
			default:
			}
		}
	}
}

func (w *windowsSource) Read(gen domain.Generation) (domain.RawPayload, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	payload := domain.RawPayload{Generation: gen}
	if !openClipboardRetry() {
		return payload, fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()

	available, _, _ := procIsClipboardFormatAvailable.Call(uintptr(cfUnicodeText))
	if available != 0 {
		if text, ok := readUnicodeText(); ok {
			payload.Text = text
			payload.OSTypes = append(payload.OSTypes, "text/plain")
		}
	}

	if cfPNG != 0 {
		available, _, _ = procIsClipboardFormatAvailable.Call(uintptr(cfPNG))
		if available != 0 {
			if data, ok := readGlobalBytes(cfPNG); ok && len(data) > 0 {
				payload.ImageBytes = data
				payload.ImageType = "png"
				payload.OSTypes = append(payload.OSTypes, "image/png")
				return payload, nil
			}
		}
	}

	available, _, _ = procIsClipboardFormatAvailable.Call(uintptr(cfDIB))
	if available != 0 {
		if data, ok := readGlobalBytes(cfDIB); ok && len(data) > 0 {
			if pngBytes, err := dibToPNG(data); err == nil && len(pngBytes) > 0 {
				payload.ImageBytes = pngBytes
				payload.ImageType = "png"
				payload.OSTypes = append(payload.OSTypes, "image/bmp")
			}
		}
	}

	if payload.Text == "" && len(payload.ImageBytes) == 0 {
		return payload, fmt.Errorf("clipboard empty")
	}
	return payload, nil
}

func readUnicodeText() (string, bool) {
	h, _, _ := procGetClipboardData.Call(uintptr(cfUnicodeText))
	if h == 0 {
		return "", false
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return "", false
	}
	defer procGlobalUnlock.Call(h)

	size, _, _ := procGlobalSize.Call(h)
	if size == 0 {
		return "", true
	}
	n := int(size) / 2
	u16 := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), n)
	for len(u16) > 0 && u16[len(u16)-1] == 0 {
		u16 = u16[:len(u16)-1]
	}
	return windows.UTF16ToString(u16), true
}

func readGlobalBytes(format uint32) ([]byte, bool) {
	h, _, _ := procGetClipboardData.Call(uintptr(format))
	if h == 0 {
		return nil, false
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return nil, false
	}
	defer procGlobalUnlock.Call(h)
	size, _, _ := procGlobalSize.Call(h)
	if size == 0 {
		return nil, false
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(size))
	out := make([]byte, len(src))
	copy(out, src)
	return out, true
}

func (w *windowsSource) Write(text string) (domain.Generation, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !openClipboardRetry() {
		return 0, fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	u16, err := windows.UTF16FromString(text)
	if err != nil {
		return 0, err
	}
	nbytes := len(u16) * 2
	h, _, err := procGlobalAlloc.Call(uintptr(gmemMoveable), uintptr(nbytes))
	if h == 0 {
		return 0, err
	}
	ptr, _, err := procGlobalLock.Call(h)
	if ptr == 0 {
		return 0, err
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(u16))
	copy(dst, u16)
	procGlobalUnlock.Call(h)

	ret, _, err := procSetClipboardData.Call(uintptr(cfUnicodeText), h)
	if ret == 0 {
		return 0, err
	}

	gen, _ := w.CurrentGeneration()
	return gen, nil
}

func (w *windowsSource) WriteImage(pngData []byte) (domain.Generation, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if len(pngData) == 0 {
		return 0, fmt.Errorf("empty png")
	}
	if cfPNG == 0 {
		return 0, fmt.Errorf("PNG clipboard format unavailable")
	}
	if !openClipboardRetry() {
		return 0, fmt.Errorf("OpenClipboard failed")
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	h, _, err := procGlobalAlloc.Call(uintptr(gmemMoveable), uintptr(len(pngData)))
	if h == 0 {
		return 0, err
	}
	ptr, _, err := procGlobalLock.Call(h)
	if ptr == 0 {
		return 0, err
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(pngData))
	copy(dst, pngData)
	procGlobalUnlock.Call(h)

	ret, _, err := procSetClipboardData.Call(uintptr(cfPNG), h)
	if ret == 0 {
		return 0, err
	}

	gen, _ := w.CurrentGeneration()
	return gen, nil
}

func openClipboardRetry() bool {
	for i := 0; i < 20; i++ {
		ret, _, _ := procOpenClipboard.Call(0)
		if ret != 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// dibToPNG converts a CF_DIB payload (BITMAPINFOHEADER + pixels) to PNG.
func dibToPNG(dib []byte) ([]byte, error) {
	if len(dib) < 40 {
		return nil, fmt.Errorf("dib too short")
	}
	biSize := binary.LittleEndian.Uint32(dib[0:4])
	if biSize < 40 {
		return nil, fmt.Errorf("bad biSize")
	}
	width := int(int32(binary.LittleEndian.Uint32(dib[4:8])))
	height := int(int32(binary.LittleEndian.Uint32(dib[8:12])))
	planes := binary.LittleEndian.Uint16(dib[12:14])
	bitCount := binary.LittleEndian.Uint16(dib[14:16])
	compression := binary.LittleEndian.Uint32(dib[16:20])
	if planes != 1 || compression != 0 {
		return nil, fmt.Errorf("unsupported dib")
	}
	if width <= 0 || height == 0 {
		return nil, fmt.Errorf("bad dimensions")
	}
	topDown := height < 0
	if topDown {
		height = -height
	}
	paletteEntries := 0
	switch bitCount {
	case 1, 4, 8:
		clrUsed := binary.LittleEndian.Uint32(dib[32:36])
		if clrUsed == 0 {
			paletteEntries = 1 << bitCount
		} else {
			paletteEntries = int(clrUsed)
		}
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("unsupported bit count %d", bitCount)
	}
	pixelOffset := int(biSize) + paletteEntries*4
	if pixelOffset >= len(dib) {
		return nil, fmt.Errorf("no pixels")
	}
	pixels := dib[pixelOffset:]
	rowBytes := ((width*int(bitCount) + 31) / 32) * 4
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		srcY := y
		if !topDown {
			srcY = height - 1 - y
		}
		row := srcY * rowBytes
		if row+rowBytes > len(pixels) {
			break
		}
		for x := 0; x < width; x++ {
			var r, g, b, a uint8
			a = 255
			switch bitCount {
			case 24:
				i := row + x*3
				b, g, r = pixels[i], pixels[i+1], pixels[i+2]
			case 32:
				i := row + x*4
				b, g, r, a = pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
				if a == 0 {
					a = 255
				}
			case 8:
				idx := int(pixels[row+x])
				if idx*4+3 >= paletteEntries*4 {
					continue
				}
				pal := dib[int(biSize)+idx*4:]
				b, g, r = pal[0], pal[1], pal[2]
			default:
				return nil, fmt.Errorf("bit count %d not decoded", bitCount)
			}
			off := y*img.Stride + x*4
			img.Pix[off] = r
			img.Pix[off+1] = g
			img.Pix[off+2] = b
			img.Pix[off+3] = a
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var _ = syscall.Errno(0)
