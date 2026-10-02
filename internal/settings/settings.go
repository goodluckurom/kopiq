package settings

import (
	"database/sql"
	"strconv"
	"sync"
)

// Defaults for M1–M3 keys.
var defaults = map[string]string{
	"capture.enabled":              "false", // stays off until onboarding completes (KQ-013)
	"onboarding.completed":         "false",
	"pause.until":                  "",
	"shortcut.quick_paste":         "CmdOrCtrl+Shift+V",
	"context.deep":                 "true",
	"sensitive.policy":             "expire",
	"sensitive.ttl_seconds":        "60",
	"retention.text_mode":          "forever",
	"retention.text_cap":           "50000",
	"retention.image_days":         "30",
	"retention.image_budget_bytes": "2147483648",
	"appearance.mode":              "system", // system | light | dark
	"appearance.accent":            "lime",   // lime | coral | sky | violet | amber | mint
	"dock.show":                    "false",
	"login.open":                   "false",
	"paste.restore_previous":       "false",
}

// Sentinel for pause-until-resume.
const PauseUntilResume = "-1"

// Settings is a typed key/value store backed by SQLite.
type Settings struct {
	db *sql.DB
	mu sync.RWMutex
}

// New wraps a database that already has a settings table.
func New(db *sql.DB) *Settings {
	return &Settings{db: db}
}

// Get returns a setting or its default.
func (s *Settings) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows || err != nil {
		if d, ok := defaults[key]; ok {
			return d
		}
		return ""
	}
	return v
}

// Set writes a setting.
func (s *Settings) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
INSERT INTO settings(key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// GetAll returns all known keys with current values.
func (s *Settings) GetAll() map[string]string {
	out := make(map[string]string, len(defaults))
	for k, d := range defaults {
		out[k] = d
	}
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		out[k] = v
	}
	return out
}

// Bool returns a boolean setting.
func (s *Settings) Bool(key string) bool {
	v := s.Get(key)
	return v == "true" || v == "1"
}

// Int returns an integer setting.
func (s *Settings) Int(key string) int {
	v := s.Get(key)
	n, _ := strconv.Atoi(v)
	return n
}

// ClearPause clears pause.until.
func (s *Settings) ClearPause() error {
	return s.Set("pause.until", "")
}

// Has reports whether a key exists in the database (ignoring defaults).
func (s *Settings) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	return err == nil
}

// MigrateOnboardingForExistingInstall marks onboarding complete when the
// database already has history from before the onboarding gate existed.
func (s *Settings) MigrateOnboardingForExistingInstall(itemCount int) error {
	if s.Has("onboarding.completed") {
		return nil
	}
	if itemCount <= 0 {
		return nil
	}
	if err := s.Set("onboarding.completed", "true"); err != nil {
		return err
	}
	if !s.Has("capture.enabled") {
		return s.Set("capture.enabled", "true")
	}
	return nil
}
