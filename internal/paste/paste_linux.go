//go:build linux

package paste

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// CaptureTarget snapshots the focused window via xdotool when available.
func CaptureTarget() (Target, error) {
	if path, _ := exec.LookPath("xdotool"); path != "" {
		out, err := exec.Command("xdotool", "getactivewindow", "getwindowpid").Output()
		if err == nil {
			pid, _ := strconv.Atoi(strings.TrimSpace(string(out)))
			name := ""
			if n, err := exec.Command("xdotool", "getactivewindow", "getwindowname").Output(); err == nil {
				name = strings.TrimSpace(string(n))
			}
			return Target{
				ProcessID:       pid,
				ApplicationName: name,
				ApplicationID:   name,
			}, nil
		}
	}
	return Target{ApplicationName: "Unknown"}, nil
}

// CanSynthesizePaste is true when a key synthesizer is installed.
func CanSynthesizePaste() bool {
	if _, err := exec.LookPath("xdotool"); err == nil {
		return true
	}
	if _, err := exec.LookPath("ydotool"); err == nil {
		return true
	}
	return false
}

// Activate focuses the previously captured window when possible.
func Activate(t Target) error {
	if t.ProcessID <= 0 {
		return nil
	}
	if _, err := exec.LookPath("xdotool"); err != nil {
		return nil
	}
	return exec.Command("xdotool", "search", "--pid", strconv.Itoa(t.ProcessID), "windowactivate").Run()
}

// SendPaste posts Ctrl+V via xdotool or ydotool.
func SendPaste() error {
	if _, err := exec.LookPath("xdotool"); err == nil {
		return exec.Command("xdotool", "key", "--clearmodifiers", "ctrl+v").Run()
	}
	if _, err := exec.LookPath("ydotool"); err == nil {
		// ydotool key: LeftCtrl down, V, LeftCtrl up (approximate)
		return exec.Command("ydotool", "key", "29:1", "47:1", "47:0", "29:0").Run()
	}
	return fmt.Errorf("paste synthesize: install xdotool or ydotool")
}

func Settle() {
	time.Sleep(40 * time.Millisecond)
}

func PromptAccessibility() bool { return CanSynthesizePaste() }

func OpenAccessibilitySettings() error {
	return fmt.Errorf("grant input automation (xdotool/ydotool) for paste-back on Linux")
}
