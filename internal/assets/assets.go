package assets

import _ "embed"

// TrayTemplate is a macOS menu-bar template icon (black + alpha) derived from design/icon.png.
//
//go:embed trayTemplate.png
var TrayTemplate []byte

// TrayIcon is a full-color 32×32 tray icon for Windows / Linux (template icons are macOS-only).
//
//go:embed trayIcon.png
var TrayIcon []byte

// AppIcon is the full-color 1024px brand mark (from design/icon.png) for Dock / about UI.
//
//go:embed appicon.png
var AppIcon []byte
