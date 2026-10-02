/** Appearance: system | light | dark + accent presets */

export type AppearanceMode = 'system' | 'light' | 'dark'

export type AccentId = 'lime' | 'coral' | 'sky' | 'violet' | 'amber' | 'mint'

export type AccentPreset = {
  id: AccentId
  label: string
  dark: string
  light: string
  onAccent: string
}

export const ACCENT_PRESETS: AccentPreset[] = [
  { id: 'lime', label: 'Lime', dark: '#a3ff12', light: '#6bb800', onAccent: '#0f1110' },
  { id: 'coral', label: 'Coral', dark: '#ff6b4a', light: '#e04a2c', onAccent: '#0f1110' },
  { id: 'sky', label: 'Sky', dark: '#4db8ff', light: '#1a7fd4', onAccent: '#0f1110' },
  { id: 'violet', label: 'Violet', dark: '#b794f6', light: '#7c3aed', onAccent: '#0f1110' },
  { id: 'amber', label: 'Amber', dark: '#f5c542', light: '#d4a017', onAccent: '#0f1110' },
  { id: 'mint', label: 'Mint', dark: '#3dd6c6', light: '#0d9488', onAccent: '#0f1110' },
]

const MODE_KEY = 'kopiq.appearance.mode'
const ACCENT_KEY = 'kopiq.appearance.accent'

export function getStoredAppearance(): AppearanceMode {
  try {
    const v = window.localStorage.getItem(MODE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    // ignore
  }
  return 'system'
}

export function getStoredAccent(): AccentId {
  try {
    const v = window.localStorage.getItem(ACCENT_KEY)
    if (ACCENT_PRESETS.some((p) => p.id === v)) return v as AccentId
  } catch {
    // ignore
  }
  return 'lime'
}

export function resolveAppearance(mode: AppearanceMode): 'light' | 'dark' {
  if (mode === 'light' || mode === 'dark') return mode
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

export function applyAccent(id: AccentId, theme?: 'light' | 'dark') {
  const preset = ACCENT_PRESETS.find((p) => p.id === id) ?? ACCENT_PRESETS[0]
  const resolved = theme ?? resolveAppearance(getStoredAppearance())
  const color = resolved === 'light' ? preset.light : preset.dark
  const root = document.documentElement
  root.style.setProperty('--lime', color)
  root.style.setProperty('--accent', color)
  root.style.setProperty('--on-lime', preset.onAccent)
  root.style.setProperty('--on-accent', preset.onAccent)
  root.style.setProperty('--lime-soft', `color-mix(in srgb, ${color} 14%, transparent)`)
  root.style.setProperty('--accent-soft', `color-mix(in srgb, ${color} 14%, transparent)`)
  try {
    window.localStorage.setItem(ACCENT_KEY, id)
  } catch {
    // ignore
  }
}

export function applyAppearance(mode: AppearanceMode, accent?: AccentId) {
  const resolved = resolveAppearance(mode)
  const root = document.documentElement
  root.dataset.theme = resolved
  root.style.colorScheme = resolved
  try {
    window.localStorage.setItem(MODE_KEY, mode)
  } catch {
    // ignore
  }
  applyAccent(accent ?? getStoredAccent(), resolved)
}

export function initAppearance() {
  applyAppearance(getStoredAppearance(), getStoredAccent())
  const mq = window.matchMedia('(prefers-color-scheme: light)')
  const onChange = () => {
    if (getStoredAppearance() === 'system') {
      applyAppearance('system', getStoredAccent())
    }
  }
  mq.addEventListener('change', onChange)
  return () => mq.removeEventListener('change', onChange)
}

/** Pull appearance from SQLite settings (shared across Library + Quick Paste windows). */
export async function syncAppearanceFromSettings(
  getSettings: () => Promise<Record<string, string>>,
): Promise<void> {
  try {
    const s = await getSettings()
    const mode = (s['appearance.mode'] as AppearanceMode) || getStoredAppearance()
    const accent = (s['appearance.accent'] as AccentId) || getStoredAccent()
    const okMode = mode === 'light' || mode === 'dark' || mode === 'system'
    const okAccent = ACCENT_PRESETS.some((p) => p.id === accent)
    applyAppearance(okMode ? mode : 'system', okAccent ? accent : 'lime')
  } catch {
    applyAppearance(getStoredAppearance(), getStoredAccent())
  }
}
