//go:build windows

package paste

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32                       = windows.NewLazySystemDLL("user32.dll")
	modKernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetForegroundWindow         = modUser32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId    = modUser32.NewProc("GetWindowThreadProcessId")
	procSetForegroundWindow         = modUser32.NewProc("SetForegroundWindow")
	procAllowSetForegroundWindow    = modUser32.NewProc("AllowSetForegroundWindow")
	procAttachThreadInput           = modUser32.NewProc("AttachThreadInput")
	procGetCurrentThreadId          = modKernel32.NewProc("GetCurrentThreadId")
	procOpenProcess                 = modKernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW  = modKernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                 = modKernel32.NewProc("CloseHandle")
	procKeybdEvent                  = modUser32.NewProc("keybd_event")
)

const (
	processQueryLimited = 0x1000
	keyeventfKeyup      = 0x0002
	vkControl           = 0x11
	vkV                 = 0x56
	asfwAny             = 0xFFFFFFFF
)

// CaptureTarget snapshots the foreground window's process.
func CaptureTarget() (Target, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return Target{}, fmt.Errorf("no foreground window")
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	t := Target{ProcessID: int(pid)}
	if path, err := processImagePath(pid); err == nil && path != "" {
		t.ApplicationID = path
		t.ApplicationName = baseName(path)
	}
	if !t.Valid() {
		return t, fmt.Errorf("no foreground process")
	}
	return t, nil
}

func processImagePath(pid uint32) (string, error) {
	h, _, err := procOpenProcess.Call(uintptr(processQueryLimited), 0, uintptr(pid))
	if h == 0 {
		return "", err
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	ret, _, err := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// CanSynthesizePaste is always true on Windows for a normal desktop app.
func CanSynthesizePaste() bool { return true }

// Activate brings the target process's main window forward (best effort).
func Activate(t Target) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if t.ProcessID <= 0 {
		return fmt.Errorf("invalid paste target")
	}
	hwnd := findWindowForPID(uint32(t.ProcessID))
	if hwnd == 0 {
		return fmt.Errorf("no window for pid %d", t.ProcessID)
	}
	procAllowSetForegroundWindow.Call(uintptr(asfwAny))

	var targetTID uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&targetTID)))
	curTID, _, _ := procGetCurrentThreadId.Call()
	if targetTID != 0 && uint32(curTID) != targetTID {
		procAttachThreadInput.Call(curTID, uintptr(targetTID), 1)
		defer procAttachThreadInput.Call(curTID, uintptr(targetTID), 0)
	}
	ret, _, err := procSetForegroundWindow.Call(hwnd)
	if ret == 0 {
		return fmt.Errorf("SetForegroundWindow: %v", err)
	}
	return nil
}

func findWindowForPID(pid uint32) uintptr {
	var found uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var wpid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&wpid)))
		if wpid == pid {
			found = hwnd
			return 0 // stop
		}
		return 1
	})
	modUser32.NewProc("EnumWindows").Call(cb, 0)
	return found
}

// SendPaste posts Ctrl+V.
func SendPaste() error {
	procKeybdEvent.Call(uintptr(vkControl), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vkV), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vkV), 0, uintptr(keyeventfKeyup), 0)
	procKeybdEvent.Call(uintptr(vkControl), 0, uintptr(keyeventfKeyup), 0)
	return nil
}

func Settle() {
	time.Sleep(40 * time.Millisecond)
}

// PromptAccessibility is a no-op on Windows (no Accessibility gate).
func PromptAccessibility() bool { return true }

// OpenAccessibilitySettings is a no-op on Windows.
func OpenAccessibilitySettings() error { return nil }
