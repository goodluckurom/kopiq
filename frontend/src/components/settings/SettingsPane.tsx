import { useCallback, useEffect, useState, type ReactNode } from 'react'
import {
  ArrowLeft,
  Keyboard,
  Shield,
  Camera,
  Info,
  Sun,
  Moon,
  Monitor,
  HardDrive,
  Clock,
  Trash2,
} from 'lucide-react'
import {
  clearDeepFields,
  clearHistory,
  cleanUpHistory,
  getSettings,
  getStorageStats,
  listExclusions,
  addExclusion,
  deleteExclusion,
  resolveApp,
  setExclusionEnabled,
  setSetting,
  type StorageBreakdown,
} from '../../lib/api'
import {
  applyAppearance,
  applyAccent,
  getStoredAppearance,
  getStoredAccent,
  ACCENT_PRESETS,
  type AppearanceMode,
  type AccentId,
} from '../../lib/appearance'
import { isLinux, isMac } from '../../lib/platform'
import { cn, formatBytes } from '../../lib/utils'
import { ConfirmModal } from '../shared/ConfirmModal'
import { AppIcon } from '../shared/AppIcon'

type Exclusion = {
  id: number
  applicationId: string
  applicationName: string
  source: string
  enabled: boolean
  iconCacheKey?: string
}

type Props = {
  onBack: () => void
}

function shortcutPresets() {
  const mac = isMac()
  return [
    {
      label: mac ? '⌘⇧V' : 'Ctrl+Shift+V',
      value: 'CmdOrCtrl+Shift+V',
    },
    {
      label: mac ? '⌘⇧C' : 'Ctrl+Shift+C',
      value: 'CmdOrCtrl+Shift+C',
    },
    {
      label: mac ? '⌥⌘V' : 'Ctrl+Alt+V',
      value: 'CmdOrCtrl+Alt+V',
    },
    {
      label: mac ? '⌃⇧Space' : 'Ctrl+Shift+Space',
      value: 'Ctrl+Shift+Space',
    },
  ]
}

export function SettingsPane({ onBack }: Props) {
  const [settings, setSettings] = useState<Record<string, string>>({})
  const [exclusions, setExclusions] = useState<Exclusion[]>([])
  const [newExName, setNewExName] = useState('')
  const [newExId, setNewExId] = useState('')
  const [exBusy, setExBusy] = useState(false)
  const [exLookupKey, setExLookupKey] = useState('')
  const [exLookupName, setExLookupName] = useState('')
  const [exLooking, setExLooking] = useState(false)
  const [exQuerySource, setExQuerySource] = useState<'id' | 'name'>('name')
  const [status, setStatus] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [appearance, setAppearance] = useState<AppearanceMode>(() => getStoredAppearance())
  const [accent, setAccent] = useState<AccentId>(() => getStoredAccent())
  const [clearConfirm, setClearConfirm] = useState(false)
  const [clearBusy, setClearBusy] = useState(false)
  const [clearDeepConfirm, setClearDeepConfirm] = useState(false)
  const [clearDeepBusy, setClearDeepBusy] = useState(false)
  const [storage, setStorage] = useState<StorageBreakdown | null>(null)

  const flash = (msg: string) => {
    setStatus(msg)
    window.setTimeout(() => setStatus(null), 2200)
  }

  useEffect(() => {
    const id = newExId.trim()
    const name = newExName.trim()
    const query =
      exQuerySource === 'id' ? id || name : name || id
    if (!query) {
      setExLookupKey('')
      setExLookupName('')
      setExLooking(false)
      return
    }
    let cancelled = false
    setExLooking(true)
    const t = window.setTimeout(() => {
      void resolveApp(query)
        .then((info) => {
          if (cancelled || !info?.found) {
            if (!cancelled && exQuerySource === 'name' && !id) {
              setExLookupKey('')
              setExLookupName('')
            }
            return
          }
          setExLookupKey(info.iconCacheKey || info.applicationId)
          if (info.applicationName) {
            setExLookupName(info.applicationName)
            if (exQuerySource === 'id') {
              setNewExName((prev) => (prev.trim() ? prev : info.applicationName))
            }
          }
          if (info.applicationId && exQuerySource === 'name') {
            setNewExId(info.applicationId)
          }
        })
        .finally(() => {
          if (!cancelled) setExLooking(false)
        })
    }, 350)
    return () => {
      cancelled = true
      window.clearTimeout(t)
    }
  }, [newExId, newExName, exQuerySource])

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const [s, ex, stats] = await Promise.all([
        getSettings(),
        listExclusions(),
        getStorageStats(),
      ])
      setSettings(s)
      setExclusions(ex)
      setStorage(stats)
      const mode = (s['appearance.mode'] as AppearanceMode) || getStoredAppearance()
      if (mode === 'light' || mode === 'dark' || mode === 'system') {
        setAppearance(mode)
      }
      const acc = (s['appearance.accent'] as AccentId) || getStoredAccent()
      if (ACCENT_PRESETS.some((p) => p.id === acc)) {
        setAccent(acc)
        applyAppearance(mode, acc)
      } else {
        applyAppearance(mode)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  const write = async (key: string, value: string) => {
    setSettings((prev) => ({ ...prev, [key]: value }))
    try {
      await setSetting(key, value)
      flash('Saved')
    } catch (e) {
      flash(e instanceof Error ? e.message : 'Save failed')
      void reload()
    }
  }

  const bool = (key: string) =>
    settings[key] === 'true' || settings[key] === '1'

  const toggle = (key: string) => {
    void write(key, bool(key) ? 'false' : 'true')
  }

  return (
    <div className="flex h-full min-w-0 flex-1 flex-col bg-[var(--charcoal)]">
      <ConfirmModal
        open={clearConfirm}
        title="Clear clipboard history?"
        description="All unpinned items will be removed. Pinned items are kept."
        confirmLabel="Clear all"
        danger
        busy={clearBusy}
        onCancel={() => {
          if (!clearBusy) setClearConfirm(false)
        }}
        onConfirm={() => {
          setClearBusy(true)
          void clearHistory()
            .then((n) => {
              flash(`Cleared ${n} items`)
              setClearConfirm(false)
            })
            .catch((e) => {
              flash(e instanceof Error ? e.message : 'Clear failed')
            })
            .finally(() => setClearBusy(false))
        }}
      />
      <ConfirmModal
        open={clearDeepConfirm}
        title="Clear window titles and URLs?"
        description="Removes deep context from existing history. Application names and times stay. New copies still follow the Deep context setting."
        confirmLabel="Clear deep fields"
        danger
        busy={clearDeepBusy}
        onCancel={() => {
          if (!clearDeepBusy) setClearDeepConfirm(false)
        }}
        onConfirm={() => {
          setClearDeepBusy(true)
          void clearDeepFields()
            .then((n) => {
              flash(`Cleared deep fields on ${n} contexts`)
              setClearDeepConfirm(false)
            })
            .catch((e) => {
              flash(e instanceof Error ? e.message : 'Clear failed')
            })
            .finally(() => setClearDeepBusy(false))
        }}
      />
      <header
        className="flex h-12 shrink-0 items-center gap-3 border-b border-[var(--border)] pr-4"
        style={{ paddingLeft: 'var(--titlebar-pad-left)' }}
      >
        <button
          type="button"
          onClick={onBack}
          className="flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-[13px] text-[var(--muted)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]"
        >
          <ArrowLeft size={15} />
          Library
        </button>
        <h1 className="text-[14px] font-semibold text-[var(--mist)]">Settings</h1>
        {status && (
          <span className="ml-auto text-[12px] text-[var(--lime)]">{status}</span>
        )}
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        {loading ? (
          <p className="text-[13px] text-[var(--subtle)]">Loading…</p>
        ) : (
          <div className="mx-auto flex max-w-[560px] flex-col gap-8">
            <Section
              icon={<Keyboard size={15} />}
              title="General"
              desc="Shortcut and appearance."
            >
              <Field label="Quick Paste shortcut">
                <div className="flex flex-wrap gap-1.5">
                  {shortcutPresets().map((p) => {
                    const active =
                      normalizeAccel(settings['shortcut.quick_paste'] || '') ===
                      normalizeAccel(p.value)
                    return (
                      <button
                        key={p.value}
                        type="button"
                        onClick={() => void write('shortcut.quick_paste', p.value)}
                        className={cn(
                          'rounded-lg border px-2.5 py-1.5 font-mono text-[12px] transition-colors',
                          active
                            ? 'border-[var(--lime)] bg-[var(--lime-soft)] text-[var(--lime)]'
                            : 'border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)] hover:bg-[var(--elevated-2)]',
                        )}
                      >
                        {p.label}
                      </button>
                    )
                  })}
                </div>
                <p className="mt-1.5 text-[11px] text-[var(--subtle)]">
                  Takes effect immediately. If another app owns the combo, tray
                  Open Quick Paste still works.
                </p>
              </Field>

              {isMac() && (
                <Toggle
                  label="Show in Dock"
                  hint="Takes effect immediately."
                  checked={bool('dock.show')}
                  onChange={() => toggle('dock.show')}
                />
              )}

              <Toggle
                label="Launch at login"
                hint="Start Kopiq when you sign in. Uses the OS login item."
                checked={bool('login.open')}
                onChange={() => toggle('login.open')}
              />

              <Field label="Appearance">
                <div className="flex flex-wrap gap-1.5">
                  {(
                    [
                      { id: 'system' as const, label: 'System', icon: Monitor },
                      { id: 'dark' as const, label: 'Dark', icon: Moon },
                      { id: 'light' as const, label: 'Light', icon: Sun },
                    ]
                  ).map((opt) => {
                    const Icon = opt.icon
                    const active = appearance === opt.id
                    return (
                      <button
                        key={opt.id}
                        type="button"
                        onClick={() => {
                          setAppearance(opt.id)
                          applyAppearance(opt.id, accent)
                          void write('appearance.mode', opt.id)
                        }}
                        className={cn(
                          'inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-[12px] font-medium',
                          active
                            ? 'border-[var(--lime)] bg-[var(--lime-soft)] text-[var(--lime)]'
                            : 'border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)]',
                        )}
                      >
                        <Icon size={13} />
                        {opt.label}
                      </button>
                    )
                  })}
                </div>
              </Field>

              <Field label="Accent color">
                <div className="flex flex-wrap gap-2">
                  {ACCENT_PRESETS.map((p) => {
                    const active = accent === p.id
                    return (
                      <button
                        key={p.id}
                        type="button"
                        title={p.label}
                        aria-label={p.label}
                        aria-pressed={active}
                        onClick={() => {
                          setAccent(p.id)
                          applyAccent(p.id)
                          void write('appearance.accent', p.id)
                        }}
                        className={cn(
                          'flex h-8 w-8 items-center justify-center rounded-full border-2 transition-transform hover:scale-105',
                          active
                            ? 'border-[var(--mist)] ring-2 ring-[var(--lime)]/40'
                            : 'border-transparent',
                        )}
                        style={{
                          background:
                            appearance === 'light' ||
                            (appearance === 'system' &&
                              window.matchMedia('(prefers-color-scheme: light)').matches)
                              ? p.light
                              : p.dark,
                        }}
                      />
                    )
                  })}
                </div>
                <p className="mt-1.5 text-[11px] text-[var(--subtle)]">
                  Used for selection, primary buttons, and highlights.
                </p>
              </Field>
            </Section>

            <Section
              icon={<Camera size={15} />}
              title="Capture"
              desc="What Kopiq records."
            >
              <Toggle
                label="Capture enabled"
                hint="Master switch. Off ignores new clipboard changes."
                checked={bool('capture.enabled')}
                onChange={() => toggle('capture.enabled')}
              />
              <Toggle
                label="Deep context"
                hint={
                  isMac()
                    ? 'Window titles, URLs, and file paths when Accessibility allows.'
                    : isLinux()
                      ? 'Window titles via xdotool when installed (X11).'
                      : 'Window titles from the foreground app when available.'
                }
                checked={bool('context.deep')}
                onChange={() => toggle('context.deep')}
              />
              <button
                type="button"
                className="rounded-lg border border-[var(--border)] bg-[var(--elevated)] px-3 py-2 text-[12px] font-medium text-[var(--mist)] hover:bg-[var(--elevated-2)]"
                onClick={() => setClearDeepConfirm(true)}
              >
                Clear window titles and URLs…
              </button>
            </Section>

            <Section
              icon={<Clock size={15} />}
              title="History"
              desc="How long copies stay. Pins are never aged out."
            >
              <Field label="Keep text">
                <p className="text-[12px] text-[var(--muted)]">
                  Forever (until the entry cap of{' '}
                  {settings['retention.text_cap'] || '50000'} unpinned text items).
                </p>
              </Field>
              <Field label="Keep images">
                <div className="flex flex-wrap gap-1.5">
                  {[7, 14, 30, 90].map((days) => {
                    const active =
                      Number(settings['retention.image_days'] || '30') === days
                    return (
                      <button
                        key={days}
                        type="button"
                        onClick={() => void write('retention.image_days', String(days))}
                        className={cn(
                          'rounded-lg border px-2.5 py-1.5 text-[12px] font-medium',
                          active
                            ? 'border-[var(--lime)] bg-[var(--lime-soft)] text-[var(--lime)]'
                            : 'border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)]',
                        )}
                      >
                        {days} days
                      </button>
                    )
                  })}
                </div>
              </Field>
              <p className="text-[11px] text-[var(--subtle)]">
                Duplicates are merged. Pinned items survive clear and retention.
              </p>
              <button
                type="button"
                className="rounded-lg border border-[var(--border)] bg-[var(--elevated)] px-3 py-2 text-[12px] font-medium text-[var(--mist)] hover:bg-[var(--elevated-2)]"
                onClick={() => setClearConfirm(true)}
              >
                Clear history (keep pins)
              </button>
            </Section>

            <Section
              icon={<HardDrive size={15} />}
              title="Storage"
              desc="What Kopiq uses on this computer."
            >
              {storage ? (
                <ul className="space-y-1.5 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[12px]">
                  <StorageRow label="Text" value={formatBytes(storage.textBytes)} />
                  <StorageRow label="Images" value={formatBytes(storage.imageBytes)} />
                  <StorageRow label="Files" value={formatBytes(storage.fileBytes)} />
                  <StorageRow label="Database" value={formatBytes(storage.databaseBytes)} />
                  <StorageRow
                    label="Total"
                    value={formatBytes(storage.totalBytes)}
                    strong
                  />
                  <li className="pt-1 text-[11px] text-[var(--subtle)]">
                    {storage.itemCount} items · {storage.pinnedCount} pinned
                  </li>
                </ul>
              ) : (
                <p className="text-[12px] text-[var(--subtle)]">
                  Storage stats need the desktop app.
                </p>
              )}
              <button
                type="button"
                className="rounded-lg bg-[var(--lime)] px-3 py-2 text-[12px] font-semibold text-[var(--on-lime)] hover:opacity-90"
                onClick={() =>
                  void cleanUpHistory()
                    .then((r) => {
                      flash(
                        r.removed
                          ? `Removed ${r.removed} · freed ${formatBytes(r.bytesFreed)}`
                          : 'Nothing to clean',
                      )
                      void reload()
                    })
                    .catch((e) =>
                      flash(e instanceof Error ? e.message : 'Clean up failed'),
                    )
                }
              >
                Clean Up
              </button>
            </Section>

            <Section
              icon={<Shield size={15} />}
              title="Privacy"
              desc="Secrets and excluded apps."
            >
              <Field label="Suspected secrets">
                <div className="flex gap-1.5">
                  {(
                    [
                      { id: 'expire', label: 'Expire in 60s' },
                      { id: 'never', label: 'Never save' },
                    ] as const
                  ).map((opt) => {
                    const active = (settings['sensitive.policy'] || 'expire') === opt.id
                    return (
                      <button
                        key={opt.id}
                        type="button"
                        onClick={() => void write('sensitive.policy', opt.id)}
                        className={cn(
                          'rounded-lg border px-3 py-1.5 text-[12px] font-medium',
                          active
                            ? 'border-[var(--lime)] bg-[var(--lime-soft)] text-[var(--lime)]'
                            : 'border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)]',
                        )}
                      >
                        {opt.label}
                      </button>
                    )
                  })}
                </div>
              </Field>

              <Field label="Excluded apps">
                <ul className="divide-y divide-[var(--border)] overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--elevated)]">
                  {exclusions.length === 0 ? (
                    <li className="px-3 py-3 text-[12px] text-[var(--subtle)]">
                      No exclusions yet.
                    </li>
                  ) : (
                    exclusions.map((ex) => (
                      <li
                        key={ex.id}
                        className="flex items-center gap-3 px-3 py-2.5"
                      >
                        <AppIcon
                          name={ex.applicationName || ex.applicationId || 'App'}
                          iconKey={ex.iconCacheKey || ex.applicationId}
                          size={22}
                        />
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-[13px] text-[var(--mist)]">
                            {ex.applicationName || ex.applicationId || 'Unknown'}
                          </p>
                          <p className="truncate font-mono text-[10px] text-[var(--subtle)]">
                            {ex.source}
                            {ex.applicationId ? ` · ${ex.applicationId}` : ''}
                          </p>
                        </div>
                        {ex.source === 'user' && (
                          <button
                            type="button"
                            title="Remove exclusion"
                            aria-label={`Remove ${ex.applicationName || ex.applicationId}`}
                            className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-[var(--subtle)] hover:bg-[var(--elevated-2)] hover:text-red-400"
                            onClick={() =>
                              void deleteExclusion(ex.id)
                                .then(() =>
                                  setExclusions((prev) => prev.filter((e) => e.id !== ex.id)),
                                )
                                .then(() => flash('Removed'))
                                .catch((e) =>
                                  flash(e instanceof Error ? e.message : 'Could not remove'),
                                )
                            }
                          >
                            <Trash2 size={14} />
                          </button>
                        )}
                        <button
                          type="button"
                          role="switch"
                          aria-checked={ex.enabled}
                          onClick={() =>
                            void setExclusionEnabled(ex.id, !ex.enabled).then(() => {
                              setExclusions((prev) =>
                                prev.map((e) =>
                                  e.id === ex.id ? { ...e, enabled: !e.enabled } : e,
                                ),
                              )
                              flash(ex.enabled ? 'Capture allowed' : 'Excluded')
                            })
                          }
                          className={cn(
                            'relative h-5 w-9 shrink-0 rounded-full transition-colors',
                            ex.enabled ? 'bg-[var(--lime)]' : 'bg-[var(--elevated-2)]',
                          )}
                        >
                          <span
                            className={cn(
                              'absolute top-0.5 h-4 w-4 rounded-full bg-[var(--on-lime)] transition-transform',
                              ex.enabled ? 'left-4' : 'left-0.5',
                            )}
                          />
                        </button>
                      </li>
                    ))
                  )}
                </ul>
                <div className="mt-2 flex flex-col gap-1.5">
                  <div className="flex items-center gap-2">
                    {(exLookupKey || newExId.trim() || (exLooking && newExName.trim())) && (
                      <AppIcon
                        name={exLookupName || newExName || newExId || 'App'}
                        iconKey={exLookupKey || newExId.trim()}
                        size={22}
                      />
                    )}
                    <input
                      value={newExName}
                      onChange={(e) => {
                        setExQuerySource('name')
                        setNewExName(e.target.value)
                      }}
                      placeholder="App name (e.g. Safari)"
                      className="min-w-0 flex-1 rounded-lg border border-[var(--border)] bg-[var(--elevated)] px-2.5 py-1.5 text-[12px] text-[var(--mist)] outline-none focus:border-[var(--lime)]"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5 sm:flex-row">
                    <input
                      value={newExId}
                      onChange={(e) => {
                        setExQuerySource('id')
                        setNewExId(e.target.value)
                      }}
                      placeholder={
                        isMac()
                          ? 'Bundle ID (optional — filled from name when found)'
                          : isLinux()
                            ? 'Desktop / WM class (optional)'
                            : 'App id / exe path (optional)'
                      }
                      className="min-w-0 flex-1 rounded-lg border border-[var(--border)] bg-[var(--elevated)] px-2.5 py-1.5 font-mono text-[11px] text-[var(--mist)] outline-none focus:border-[var(--lime)]"
                    />
                    <button
                      type="button"
                      disabled={
                        exBusy ||
                        exLooking ||
                        (!newExName.trim() && !newExId.trim())
                      }
                      className="shrink-0 rounded-lg bg-[var(--elevated-2)] px-3 py-1.5 text-[12px] font-medium text-[var(--mist)] disabled:opacity-50"
                      onClick={() => {
                        const id = newExId.trim()
                        const name = newExName.trim() || exLookupName.trim()
                        if (!name && !id) return
                        setExBusy(true)
                        void addExclusion(id, name)
                          .then(() => listExclusions())
                          .then((rows) => {
                            setExclusions(rows)
                            setNewExName('')
                            setNewExId('')
                            setExLookupKey('')
                            setExLookupName('')
                            setExQuerySource('name')
                            flash('Exclusion added')
                          })
                          .catch((e) =>
                            flash(e instanceof Error ? e.message : 'Could not add'),
                          )
                          .finally(() => setExBusy(false))
                      }}
                    >
                      {exLooking ? 'Looking up…' : 'Add'}
                    </button>
                  </div>
                </div>
                <p className="mt-1.5 text-[11px] text-[var(--subtle)]">
                  Type an app name or bundle id to fetch its icon. User-added apps
                  can be removed entirely; builtins can only be toggled off.
                </p>
              </Field>
            </Section>

            <Section icon={<Info size={15} />} title="About" desc="Kopiq is local-first.">
              <p className="text-[13px] text-[var(--muted)]">
                Clipboard history stays on this computer. No account. No telemetry.
              </p>
            </Section>
          </div>
        )}
      </div>
    </div>
  )
}

function StorageRow({
  label,
  value,
  strong,
}: {
  label: string
  value: string
  strong?: boolean
}) {
  return (
    <li
      className={cn(
        'flex items-center justify-between gap-3',
        strong && 'border-t border-[var(--border)] pt-1.5 font-semibold text-[var(--mist)]',
      )}
    >
      <span className={strong ? undefined : 'text-[var(--muted)]'}>{label}</span>
      <span className="tabular-nums text-[var(--mist)]">{value}</span>
    </li>
  )
}

function normalizeAccel(s: string) {
  return s.replace(/Command\+/gi, 'CmdOrCtrl+').replace(/Cmd\+/gi, 'CmdOrCtrl+').toLowerCase()
}

function Section({
  icon,
  title,
  desc,
  children,
}: {
  icon: ReactNode
  title: string
  desc: string
  children: ReactNode
}) {
  return (
    <section className="space-y-3">
      <div className="flex items-start gap-2.5">
        <span className="mt-0.5 text-[var(--muted)]">{icon}</span>
        <div>
          <h2 className="text-[14px] font-semibold text-[var(--mist)]">{title}</h2>
          <p className="text-[12px] text-[var(--subtle)]">{desc}</p>
        </div>
      </div>
      <div className="space-y-4 border-l border-[var(--border)] pl-5">{children}</div>
    </section>
  )
}

function Field({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div>
      <p className="mb-1.5 text-[12px] font-medium text-[var(--mist)]">{label}</p>
      {children}
    </div>
  )
}

function Toggle({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string
  hint?: string
  checked: boolean
  onChange: () => void
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="min-w-0">
        <p className="text-[12px] font-medium text-[var(--mist)]">{label}</p>
        {hint && <p className="text-[11px] text-[var(--subtle)]">{hint}</p>}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={onChange}
        className={cn(
          'relative h-5 w-9 shrink-0 rounded-full transition-colors',
          checked ? 'bg-[var(--lime)]' : 'bg-[var(--elevated-2)]',
        )}
      >
        <span
          className={cn(
            'absolute top-0.5 h-4 w-4 rounded-full bg-[var(--on-lime)] transition-transform',
            checked ? 'left-4' : 'left-0.5',
          )}
        />
      </button>
    </div>
  )
}
