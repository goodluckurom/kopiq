//go:build linux || freebsd || openbsd || netbsd

package context

import (
	"os/exec"
	"strings"
	"time"

	"kopiq/internal/domain"
)

type linuxProvider struct{}

// New returns the platform context provider.
func New() Provider {
	return &linuxProvider{}
}

func (l *linuxProvider) Capture(deep bool) (domain.Snapshot, error) {
	snap := domain.Snapshot{
		ApplicationName: "Unknown",
		CapturedAt:      time.Now().UnixMilli(),
		Deep:            false,
	}

	if _, err := exec.LookPath("xdotool"); err != nil {
		return snap, nil
	}

	if name, err := exec.Command("xdotool", "getactivewindow", "getwindowname").Output(); err == nil {
		n := strings.TrimSpace(string(name))
		if n != "" {
			snap.ApplicationName = n
			snap.ApplicationID = n
			if deep {
				snap.WindowTitle = n
				snap.Deep = true
			}
		}
	}
	if class, err := exec.Command("xdotool", "getactivewindow", "getwindowclassname").Output(); err == nil {
		c := strings.TrimSpace(string(class))
		if c != "" {
			snap.ApplicationID = c
			if snap.ApplicationName == "Unknown" || snap.ApplicationName == "" {
				snap.ApplicationName = c
			}
		}
	}
	return snap, nil
}
