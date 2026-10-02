//go:build darwin

package normalize

import (
	"os/exec"
	"strings"
)

func macScreenshotLocation() string {
	out, err := exec.Command("defaults", "read", "com.apple.screencapture", "location").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
