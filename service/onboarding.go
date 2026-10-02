package service

import (
	"os"
	"strings"

	"kopiq/internal/icons"
	"kopiq/internal/paste"
)

// FrontmostAppDTO is a lightweight app identity for onboarding exclusions.
type FrontmostAppDTO struct {
	ApplicationID   string `json:"applicationId"`
	ApplicationName string `json:"applicationName"`
}

// IsDev reports whether we're running against the Vite frontend dev server.
func IsDev() bool {
	return os.Getenv("FRONTEND_DEVSERVER_URL") != ""
}

// NeedsOnboarding is true until the user finishes the first-run gate.
// In dev it is always true so every refresh / relaunch re-runs the flow.
func (s *SettingsService) NeedsOnboarding() bool {
	if IsDev() {
		return true
	}
	return !s.Settings.Bool("onboarding.completed")
}

// CompleteOnboarding enables capture and marks onboarding done.
// In dev, capture is enabled for the session but completion is not persisted,
// so the next refresh still shows onboarding.
func (s *SettingsService) CompleteOnboarding() error {
	if err := s.Settings.Set("capture.enabled", "true"); err != nil {
		return err
	}
	if !IsDev() {
		if err := s.Settings.Set("onboarding.completed", "true"); err != nil {
			return err
		}
	}
	if s.OnSet != nil {
		_ = s.OnSet("capture.enabled", "true")
		if !IsDev() {
			_ = s.OnSet("onboarding.completed", "true")
		}
	}
	return nil
}

// AccessibilityGranted reports whether Accessibility is trusted (macOS).
func (s *SettingsService) AccessibilityGranted() bool {
	return paste.CanSynthesizePaste()
}

// OpenAccessibilitySettings opens the OS Accessibility pane and may prompt.
func (s *SettingsService) OpenAccessibilitySettings() error {
	_ = paste.PromptAccessibility()
	return paste.OpenAccessibilitySettings()
}

// LaunchFrontmost returns the app that was frontmost at launch (for exclusions).
func (s *SettingsService) LaunchFrontmost() FrontmostAppDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.launchFrontmost
}

// ExcludeLaunchFrontmost adds the pre-launch frontmost app to exclusions.
func (s *SettingsService) ExcludeLaunchFrontmost() error {
	s.mu.RLock()
	app := s.launchFrontmost
	s.mu.RUnlock()
	if app.ApplicationID == "" && app.ApplicationName == "" {
		return nil
	}
	if isSelfApp(app.ApplicationID, app.ApplicationName) {
		return nil
	}
	if s.Store == nil {
		return nil
	}
	if app.ApplicationID != "" {
		_, _ = icons.Ensure(app.ApplicationID)
	}
	return s.Store.AddExclusion(app.ApplicationID, app.ApplicationName)
}

// SetLaunchFrontmost records the app that was active before the library opened.
func (s *SettingsService) SetLaunchFrontmost(id, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if isSelfApp(id, name) {
		s.launchFrontmost = FrontmostAppDTO{}
		return
	}
	s.launchFrontmost = FrontmostAppDTO{
		ApplicationID:   id,
		ApplicationName: name,
	}
}

func isSelfApp(id, name string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	name = strings.ToLower(strings.TrimSpace(name))
	if strings.Contains(id, "kopiq") || name == "kopiq" {
		return true
	}
	return false
}
