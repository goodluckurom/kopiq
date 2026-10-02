package main

import (
	"embed"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"kopiq/internal/appdata"
	"kopiq/internal/assets"
	"kopiq/internal/capture"
	"kopiq/internal/clipboard"
	appctx "kopiq/internal/context"
	"kopiq/internal/dock"
	"kopiq/internal/domain"
	"kopiq/internal/retention"
	"kopiq/internal/screenshots"
	"kopiq/internal/settings"
	"kopiq/internal/stack"
	"kopiq/internal/store"
	"kopiq/service"
)

//go:embed all:frontend/dist
var assetsFS embed.FS

func init() {
	application.RegisterEvent[domain.HistoryChangedEvent]("history.changed")
	application.RegisterEvent[domain.CaptureStatus]("capture.state")
	application.RegisterEvent[string]("quickpaste.opened")
	application.RegisterEvent[string]("quickpaste.toast")
	application.RegisterEvent[string]("appearance.changed")
}

func main() {
	dbPath, err := appdata.DBPath()
	if err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	set := settings.New(st.DB())
	_ = set.ClearPause() // pause does not survive restart (KQ-009)
	devMode := os.Getenv("FRONTEND_DEVSERVER_URL") != ""
	if devMode {
		// Dev: always re-gate capture so each launch/refresh exercises onboarding.
		_ = set.Set("capture.enabled", "false")
	} else if n, err := st.CountItems(); err == nil {
		if err := set.MigrateOnboardingForExistingInstall(n); err != nil {
			log.Printf("onboarding migrate: %v", err)
		}
	}

	src := clipboard.New()
	ctxProvider := appctx.New()

	var app *application.App
	var pipeline *capture.Pipeline
	var shotWatcher *screenshots.Watcher

	history := &service.HistoryService{Store: st}
	search := &service.SearchService{Store: st}
	captureSvc := &service.CaptureService{}
	settingsSvc := &service.SettingsService{Settings: set, Store: st}
	quickPasteSvc := &service.QuickPasteService{Store: st, Settings: set}
	organizeSvc := &service.OrganizeService{Store: st}
	transformSvc := &service.TransformService{Store: st}
	stackSession := &stack.Session{}
	quickPasteSvc.Stack = stackSession
	stackSvc := &service.StackService{Store: st, Session: stackSession}

	macPolicy := application.ActivationPolicyAccessory
	if set.Bool("dock.show") || os.Getenv("FRONTEND_DEVSERVER_URL") != "" {
		macPolicy = application.ActivationPolicyRegular
	}

	emitHistory := func(evt domain.HistoryChangedEvent) {
		if app != nil {
			app.Event.Emit("history.changed", evt)
		}
	}
	history.Emit = emitHistory
	organizeSvc.Emit = emitHistory

	app = application.New(application.Options{
		Name:        "Kopiq",
		Description: "Local-first developer clipboard manager",
		Icon:        assets.AppIcon,
		Services: []application.Service{
			application.NewService(history),
			application.NewService(search),
			application.NewService(captureSvc),
			application.NewService(settingsSvc),
			application.NewService(quickPasteSvc),
			application.NewService(organizeSvc),
			application.NewService(transformSvc),
			application.NewService(stackSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assetsFS),
		},
		Mac: application.MacOptions{
			ActivationPolicy: macPolicy,
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	pipeline = capture.New(src, ctxProvider, st, set, func(evt domain.HistoryChangedEvent) {
		app.Event.Emit("history.changed", evt)
	})
	history.Pipeline = pipeline
	captureSvc.Pipeline = pipeline
	quickPasteSvc.Pipeline = pipeline
	transformSvc.Pipeline = pipeline

	history.SaveDialog = func(defaultName string) (string, error) {
		return app.Dialog.SaveFile().
			SetFilename(defaultName).
			AddFilter("Images", "*.png;*.jpg;*.jpeg").
			PromptForSingleSelection()
	}
	history.RevealPath = revealInFileManager
	history.OpenPath = openInDefaultApp

	// Snapshot frontmost before any Kopiq window steals focus (onboarding exclusions).
	if snap, err := ctxProvider.Capture(false); err == nil {
		settingsSvc.SetLaunchFrontmost(snap.ApplicationID, snap.ApplicationName)
	}

	// Sensitive expiry loop (KQ-008).
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for range t.C {
			n, err := st.PurgeExpired(time.Now().UnixMilli())
			if err != nil {
				log.Printf("purge expired: %v", err)
				continue
			}
			if n > 0 && app != nil {
				app.Event.Emit("history.changed", domain.HistoryChangedEvent{Action: "delete"})
			}
		}
	}()

	// Retention: age/cap/budget (KQ-010). Pins are never deleted.
	ret := &retention.Runner{
		Store:    st,
		Settings: set,
		Emit: func() {
			if app != nil {
				app.Event.Emit("history.changed", domain.HistoryChangedEvent{Action: "delete"})
			}
		},
	}
	ret.Start()

	library := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "library",
		Title:            "Kopiq",
		Width:            1280,
		Height:           800,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(0x0F, 0x11, 0x10),
		URL:              "/",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})
	library.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		library.Hide()
		e.Cancel()
	})

	qpOpts := application.WebviewWindowOptions{
		Name:        "quick-paste",
		Title:       "Quick Paste",
		Width:       640,
		Height:      480,
		MinWidth:    480,
		MinHeight:   360,
		Frameless:   true,
		AlwaysOnTop: true,
		Hidden:      true,
		URL:         "/#/quick-paste",
		Windows: application.WindowsWindow{
			HiddenOnTaskbar:        true,
			NonClientRegionSupport: true,
		},
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTransparent,
		},
	}
	// Transparent chrome is a macOS/Vibrancy pattern. On Windows use a solid
	// surface so the frameless panel paints reliably, and enable drag regions.
	if runtime.GOOS == "windows" {
		qpOpts.BackgroundType = application.BackgroundTypeSolid
		qpOpts.BackgroundColour = application.NewRGB(0x0F, 0x11, 0x10)
	} else {
		qpOpts.BackgroundType = application.BackgroundTypeTransparent
		qpOpts.BackgroundColour = application.NewRGBA(0, 0, 0, 0)
	}
	quickPaste := app.Window.NewWithOptions(qpOpts)
	quickPaste.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		quickPaste.Hide()
		e.Cancel()
	})

	quickPasteSvc.Show = func() {
		quickPaste.Show()
		quickPaste.Focus()
		app.Event.Emit("quickpaste.opened", "")
	}
	quickPasteSvc.Hide = func() {
		quickPaste.Hide()
	}

	openQuickPaste := func() {
		_ = quickPasteSvc.Prepare()
	}

	shortcut := normalizeShortcut(set.Get("shortcut.quick_paste"))
	currentShortcut := shortcut
	if err := app.GlobalShortcut.Register(shortcut, openQuickPaste); err != nil {
		log.Printf("global shortcut %q: %v (tray Open Quick Paste still works)", shortcut, err)
	}

	applyLoginOpen := func(enabled bool) error {
		if app == nil || app.Autostart == nil {
			return nil
		}
		if enabled {
			return app.Autostart.Enable()
		}
		return app.Autostart.Disable()
	}

	settingsSvc.OnSet = func(key, value string) error {
		switch key {
		case "capture.enabled", "onboarding.completed":
			if pipeline != nil && app != nil {
				app.Event.Emit("capture.state", pipeline.Status())
			}
			return nil
		case "dock.show":
			dock.SetVisible(value == "true" || value == "1")
			return nil
		case "login.open":
			return applyLoginOpen(value == "true" || value == "1")
		case "appearance.mode", "appearance.accent":
			if app != nil {
				app.Event.Emit("appearance.changed", value)
			}
			return nil
		case "shortcut.quick_paste":
			next := normalizeShortcut(value)
			if next == currentShortcut {
				return nil
			}
			if currentShortcut != "" {
				_ = app.GlobalShortcut.Unregister(currentShortcut)
			}
			if err := app.GlobalShortcut.Register(next, openQuickPaste); err != nil {
				// Best-effort restore previous binding.
				_ = app.GlobalShortcut.Register(currentShortcut, openQuickPaste)
				return err
			}
			currentShortcut = next
			return nil
		default:
			return nil
		}
	}

	// Sync OS login item with the stored preference (no-op if unsupported).
	if err := applyLoginOpen(set.Bool("login.open")); err != nil {
		log.Printf("login.open sync: %v", err)
	}

	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(assets.TrayTemplate)
	} else {
		tray.SetIcon(assets.TrayIcon)
	}
	tray.SetTooltip("Kopiq")

	var lastTrayStack string
	var rebuildTrayMenu func()
	rebuildTrayMenu = func() {
		st := stackSession.Status()
		label := ""
		if st.Armed {
			label = st.Label
		}

		menu := app.Menu.New()
		menu.Add("Open Library").OnClick(func(ctx *application.Context) {
			library.Show()
			library.Focus()
		})
		menu.Add("Open Quick Paste").OnClick(func(ctx *application.Context) {
			openQuickPaste()
		})
		menu.Add("Pinned").OnClick(func(ctx *application.Context) {
			library.Show()
			library.Focus()
			library.SetURL("/#/")
			library.ExecJS(`window.dispatchEvent(new Event('kopiq:open-pinned'))`)
		})
		menu.Add("Settings").OnClick(func(ctx *application.Context) {
			library.Show()
			library.Focus()
			library.SetURL("/#/settings")
			library.ExecJS(`window.dispatchEvent(new Event('kopiq:open-settings'))`)
		})
		if st.Armed {
			menu.AddSeparator()
			menu.Add(st.Label).SetEnabled(false)
			menu.Add("Stop stack").OnClick(func(ctx *application.Context) {
				stackSession.Disarm()
				lastTrayStack = "\x00" // force rebuild
				rebuildTrayMenu()
			})
			tray.SetTooltip("Kopiq · " + st.Label)
		} else {
			tray.SetTooltip("Kopiq")
		}
		menu.AddSeparator()
		menu.Add("Pause 5 minutes").OnClick(func(ctx *application.Context) {
			_ = pipeline.Pause(5)
			app.Event.Emit("capture.state", pipeline.Status())
		})
		menu.Add("Pause 15 minutes").OnClick(func(ctx *application.Context) {
			_ = pipeline.Pause(15)
			app.Event.Emit("capture.state", pipeline.Status())
		})
		menu.Add("Pause 1 hour").OnClick(func(ctx *application.Context) {
			_ = pipeline.Pause(60)
			app.Event.Emit("capture.state", pipeline.Status())
		})
		menu.Add("Pause until resumed").OnClick(func(ctx *application.Context) {
			_ = pipeline.Pause(0)
			app.Event.Emit("capture.state", pipeline.Status())
		})
		menu.Add("Resume").OnClick(func(ctx *application.Context) {
			_ = pipeline.Resume()
			app.Event.Emit("capture.state", pipeline.Status())
		})
		menu.AddSeparator()
		menu.Add("Clear History").OnClick(func(ctx *application.Context) {
			if _, err := history.ClearHistory(); err != nil {
				log.Printf("clear history: %v", err)
				return
			}
			app.Event.Emit("history.changed", domain.HistoryChangedEvent{Action: "delete"})
		})
		menu.AddSeparator()
		menu.Add("Quit").OnClick(func(ctx *application.Context) {
			if shotWatcher != nil {
				shotWatcher.Stop()
			}
			pipeline.Stop()
			app.Quit()
		})
		tray.SetMenu(menu)
		lastTrayStack = label
	}
	rebuildTrayMenu()
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for range t.C {
			st := stackSession.Status()
			want := ""
			if st.Armed {
				want = st.Label
			}
			if want != lastTrayStack {
				rebuildTrayMenu()
			}
		}
	}()

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		library.Show()
		library.Focus()
		if settingsSvc.NeedsOnboarding() {
			library.SetURL("/#/onboarding")
			library.ExecJS(`window.dispatchEvent(new Event('kopiq:open-onboarding'))`)
		}
	})

	if err := pipeline.Start(); err != nil {
		log.Printf("capture start: %v", err)
	}

	shotWatcher = screenshots.New(func(path string) {
		if err := pipeline.IngestImageFile(path); err != nil {
			log.Printf("screenshot ingest %s: %v", path, err)
		}
	})
	shotWatcher.OnDetect(pipeline.NoteScreenshotAppeared)
	shotWatcher.Start()
	defer shotWatcher.Stop()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func normalizeShortcut(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "CmdOrCtrl+Shift+V"
	}
	out := s
	out = strings.ReplaceAll(out, "Command+", "Cmd+")
	out = strings.ReplaceAll(out, "Control+", "Ctrl+")
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "command+") || strings.HasPrefix(lower, "cmd+") {
		if i := strings.Index(out, "+"); i >= 0 {
			return "CmdOrCtrl" + out[i:]
		}
	}
	return out
}

func revealInFileManager(absPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", absPath).Start()
	case "windows":
		return exec.Command("explorer", "/select,", absPath).Start()
	default:
		return exec.Command("xdg-open", filepathDir(absPath)).Start()
	}
}

func openInDefaultApp(absPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", absPath).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", absPath).Start()
	default:
		return exec.Command("xdg-open", absPath).Start()
	}
}

func filepathDir(p string) string {
	i := strings.LastIndexAny(p, `/\`)
	if i < 0 {
		return "."
	}
	return p[:i]
}
