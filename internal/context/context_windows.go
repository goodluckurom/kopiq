//go:build windows

package context

import (
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"kopiq/internal/domain"
	"kopiq/internal/icons"
)

var (
	modUser32                       = windows.NewLazySystemDLL("user32.dll")
	modKernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetForegroundWindow         = modUser32.NewProc("GetForegroundWindow")
	procGetWindowTextW              = modUser32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId    = modUser32.NewProc("GetWindowThreadProcessId")
	procOpenProcess                 = modKernel32.NewProc("OpenProcess")
	procCloseHandle                 = modKernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW  = modKernel32.NewProc("QueryFullProcessImageNameW")
)

const processQueryLimitedInformation = 0x1000

type windowsProvider struct{}

// New returns the platform context provider.
func New() Provider {
	return &windowsProvider{}
}

func (w *windowsProvider) Capture(deep bool) (domain.Snapshot, error) {
	snap := domain.Snapshot{
		ApplicationName: "Unknown",
		CapturedAt:      time.Now().UnixMilli(),
		Deep:            false,
	}

	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return snap, nil
	}

	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	snap.ProcessID = int(pid)

	if path, err := processImagePath(pid); err == nil && path != "" {
		snap.ApplicationID = path
		snap.ApplicationName = trimExe(filepath.Base(path))
		if key, err := icons.Ensure(path); err == nil && key != "" {
			snap.IconCacheKey = key
		} else {
			snap.IconCacheKey = icons.KeyFromAppID(path)
		}
	}

	if deep {
		if title := windowTitle(hwnd); title != "" {
			snap.WindowTitle = title
			snap.Deep = true
		}
	}
	return snap, nil
}

func trimExe(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return name[:len(name)-len(ext)]
}

func windowTitle(hwnd uintptr) string {
	buf := make([]uint16, 512)
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

func processImagePath(pid uint32) (string, error) {
	if pid == 0 {
		return "", syscall.EINVAL
	}
	h, _, err := procOpenProcess.Call(uintptr(processQueryLimitedInformation), 0, uintptr(pid))
	if h == 0 {
		return "", err
	}
	defer procCloseHandle.Call(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	ret, _, err := procQueryFullProcessImageNameW.Call(
		h,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}
