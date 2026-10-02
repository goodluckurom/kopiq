import { System } from '@wailsio/runtime'

/** True when running inside the Wails desktop shell on macOS. */
export function isMac(): boolean {
  try {
    return System.IsMac()
  } catch {
    return false
  }
}

/** True when running inside the Wails desktop shell on Windows. */
export function isWindows(): boolean {
  try {
    return System.IsWindows()
  } catch {
    return false
  }
}

/** True when running inside the Wails desktop shell on Linux. */
export function isLinux(): boolean {
  try {
    return System.IsLinux()
  } catch {
    return false
  }
}

/** Platform modifier glyph for keyboard hints (⌘ vs Ctrl). */
export function modGlyph(): string {
  return isMac() ? '⌘' : 'Ctrl'
}

/** Search / command palette shortcut label. */
export function searchShortcutLabel(): string {
  return isMac() ? '⌘K' : 'Ctrl+K'
}

/** Tray / status-area label for user-facing copy. */
export function trayLabel(): string {
  return isMac() ? 'menu bar' : 'system tray'
}

/** Apply `data-os` on <html> so CSS can key off the host platform. */
export function applyPlatformAttr(): void {
  if (typeof document === 'undefined') return
  const os = isMac() ? 'macos' : isWindows() ? 'windows' : isLinux() ? 'linux' : 'other'
  document.documentElement.dataset.os = os
}
