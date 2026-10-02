package retention

import (
	"log"
	"strconv"
	"time"

	"kopiq/internal/settings"
	"kopiq/internal/store"
)

// Runner applies age/cap/budget rules on a schedule.
type Runner struct {
	Store    *store.Store
	Settings *settings.Settings
	Emit     func()
}

// Start runs retention on boot and every 10 minutes.
func (r *Runner) Start() {
	go func() {
		r.Run("startup")
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			r.Run("tick")
		}
	}()
}

// Run executes one retention pass. reason is for logs.
func (r *Runner) Run(reason string) store.RetentionResult {
	cfg := r.config()
	res, err := r.Store.ApplyRetention(cfg)
	if err != nil {
		log.Printf("retention (%s): %v", reason, err)
		return res
	}
	if res.Removed > 0 {
		log.Printf("retention (%s): removed %d items, freed ~%d bytes", reason, res.Removed, res.BytesFreed)
		if r.Emit != nil {
			r.Emit()
		}
	}
	return res
}

func (r *Runner) config() store.RetentionConfig {
	cfg := store.RetentionConfig{
		TextCap:          50000,
		ImageDays:        30,
		ImageBudgetBytes: 2 << 30,
		Now:              time.Now().UnixMilli(),
	}
	if r.Settings == nil {
		return cfg
	}
	if n := r.Settings.Int("retention.text_cap"); n > 0 {
		cfg.TextCap = n
	}
	if n := r.Settings.Int("retention.image_days"); n >= 0 {
		cfg.ImageDays = n
	}
	if raw := r.Settings.Get("retention.image_budget_bytes"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			cfg.ImageBudgetBytes = n
		}
	}
	return cfg
}
