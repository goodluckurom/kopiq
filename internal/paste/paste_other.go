//go:build !darwin && !windows && !linux

package paste

import "fmt"

// CaptureTarget is not implemented on this platform yet.
func CaptureTarget() (Target, error) {
	return Target{}, fmt.Errorf("paste target capture not implemented")
}

func CanSynthesizePaste() bool { return false }

func Activate(t Target) error {
	return fmt.Errorf("paste activate not implemented")
}

func SendPaste() error {
	return fmt.Errorf("paste synthesize not implemented")
}

func Settle() {}

func PromptAccessibility() bool { return false }

func OpenAccessibilitySettings() error {
	return fmt.Errorf("accessibility settings not available on this platform")
}
