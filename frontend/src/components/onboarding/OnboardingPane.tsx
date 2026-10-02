import { useCallback, useEffect, useMemo, useState, type CSSProperties, type ReactNode } from 'react'
import {
  Shield,
  Camera,
  Eye,
  Ban,
  KeyRound,
  Check,
  Moon,
  Sun,
  Monitor,
} from 'lucide-react'
import {
  accessibilityGranted,
  completeOnboarding,
  excludeLaunchFrontmost,
  getLaunchFrontmost,
  getSettings,
  listExclusions,
  openAccessibilitySettings,
  setSetting,
  type ExclusionRow,
  type FrontmostApp,
} from '../../lib/api'
import {
  applyAppearance,
  getStoredAppearance,
  type AppearanceMode,
} from '../../lib/appearance'
import { isMac, trayLabel } from '../../lib/platform'
import { cn } from '../../lib/utils'
import { AppIcon } from '../shared/AppIcon'

type Props = {
  onDone: () => void
}

const ALL_STEPS = [
  'stores',
  'theme',
  'deep',
  'sensitive',
  'exclusions',
  'permissions',
  'enable',
] as const

type StepId = (typeof ALL_STEPS)[number]

const STEP_IMAGES: Partial<Record<StepId, string>> = {
  stores: '/brand/onboarding/onboard-stores.jpg',
  theme: '/brand/onboarding/onboard-theme.jpg',
  deep: '/brand/onboarding/onboard-deep-context.jpg',
  sensitive: '/brand/onboarding/onboard-privacy.jpg',
  exclusions: '/brand/onboarding/onboard-privacy.jpg',
}

export function OnboardingPane({ onDone }: Props) {
  const [stepIdx, setStepIdx] = useState(0)
  const [deep, setDeep] = useState(true)
  const [policy, setPolicy] = useState<'expire' | 'never'>('expire')
  const [theme, setTheme] = useState<AppearanceMode>(() => getStoredAppearance())
  const [exclusions, setExclusions] = useState<ExclusionRow[]>([])
  const [frontmost, setFrontmost] = useState<FrontmostApp | null>(null)
  const [excludedLaunch, setExcludedLaunch] = useState(false)
  const [axOk, setAxOk] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const steps = useMemo(
    () => (isMac() ? ALL_STEPS : ALL_STEPS.filter((s) => s !== 'permissions')),
    [],
  )
  const step = steps[stepIdx]
  const hero = step ? STEP_IMAGES[step] : undefined

  const refreshAx = useCallback(async () => {
    setAxOk(await accessibilityGranted())
  }, [])

  useEffect(() => {
    void (async () => {
      const [s, ex, fm] = await Promise.all([
        getSettings(),
        listExclusions(),
        getLaunchFrontmost(),
      ])
      setDeep(s['context.deep'] !== 'false' && s['context.deep'] !== '0')
      const p = s['sensitive.policy']
      setPolicy(p === 'never' ? 'never' : 'expire')
      const mode = (s['appearance.mode'] as AppearanceMode) || getStoredAppearance()
      if (mode === 'light' || mode === 'dark' || mode === 'system') {
        setTheme(mode)
        applyAppearance(mode)
      }
      setExclusions(ex.filter((e) => e.enabled))
      if (fm?.applicationId || fm?.applicationName) setFrontmost(fm)
      await refreshAx()
    })()
  }, [refreshAx])

  useEffect(() => {
    if (step !== 'permissions') return
    const t = window.setInterval(() => void refreshAx(), 1500)
    return () => window.clearInterval(t)
  }, [step, refreshAx])

  const chooseTheme = (mode: AppearanceMode) => {
    setTheme(mode)
    applyAppearance(mode)
    void setSetting('appearance.mode', mode)
  }

  const next = async () => {
    setError(null)
    try {
      if (step === 'theme') {
        await setSetting('appearance.mode', theme)
      }
      if (step === 'deep') {
        await setSetting('context.deep', deep ? 'true' : 'false')
      }
      if (step === 'sensitive') {
        await setSetting('sensitive.policy', policy)
        if (policy === 'expire') await setSetting('sensitive.ttl_seconds', '60')
      }
      if (step === 'enable') {
        setBusy(true)
        await completeOnboarding()
        onDone()
        return
      }
      setStepIdx((i) => Math.min(i + 1, steps.length - 1))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  const back = () => {
    setError(null)
    setStepIdx((i) => Math.max(i - 1, 0))
  }

  return (
    <div className="relative flex h-full min-w-0 flex-1 flex-col bg-[var(--charcoal)]">
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.35]"
        style={{
          background:
            'radial-gradient(ellipse 80% 50% at 50% -10%, rgb(163 255 18 / 0.12), transparent 55%)',
        }}
      />

      <header
        className="relative z-[1] flex shrink-0 items-center justify-between border-b border-[var(--border)] pr-6"
        style={
          {
            paddingTop: 'var(--titlebar-pad-top)',
            paddingLeft: 'var(--titlebar-pad-left)',
            paddingBottom: '12px',
            WebkitAppRegion: 'drag',
          } as CSSProperties
        }
      >
        <div style={{ WebkitAppRegion: 'no-drag' } as CSSProperties}>
          <p className="text-[11px] font-medium uppercase tracking-[0.14em] text-[var(--subtle)]">
            Setup
          </p>
          <h1 className="text-[15px] font-semibold text-[var(--mist)]">Kopiq</h1>
        </div>
        <p className="font-mono text-[11px] text-[var(--subtle)]">
          {stepIdx + 1} / {steps.length}
        </p>
      </header>

      <div className="relative z-[1] mx-auto flex w-full max-w-[640px] min-h-0 flex-1 flex-col px-6 py-6">
        <div className="mb-5 flex gap-1.5">
          {steps.map((id, i) => (
            <span
              key={id}
              className={cn(
                'h-1 flex-1 rounded-full transition-colors',
                i <= stepIdx ? 'bg-[var(--lime)]' : 'bg-[var(--elevated-2)]',
              )}
            />
          ))}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto pb-4">
          {hero && (
            <div className="mb-5 overflow-hidden rounded-[14px] border border-[var(--border)] bg-[var(--elevated)] shadow-[0_20px_50px_rgb(0_0_0_/0.25)]">
              <img
                src={hero}
                alt=""
                className="aspect-[16/9] w-full object-cover object-center"
                draggable={false}
              />
            </div>
          )}

          {step === 'stores' && (
            <StepBlock
              icon={<Camera size={18} />}
              title="What Kopiq stores"
              body="Text, images, and screenshots you copy — plus the app they came from. Everything stays on this computer. No account. No cloud."
            />
          )}

          {step === 'theme' && (
            <StepBlock
              icon={<Sun size={18} />}
              title="Appearance"
              body="Pick how Kopiq looks. You can change this later in Settings."
            >
              <div className="grid grid-cols-3 gap-2">
                {(
                  [
                    { id: 'system' as const, label: 'System', icon: Monitor },
                    { id: 'dark' as const, label: 'Dark', icon: Moon },
                    { id: 'light' as const, label: 'Light', icon: Sun },
                  ]
                ).map((opt) => {
                  const Icon = opt.icon
                  const active = theme === opt.id
                  return (
                    <button
                      key={opt.id}
                      type="button"
                      onClick={() => chooseTheme(opt.id)}
                      className={cn(
                        'flex flex-col items-center gap-2 rounded-[12px] border px-3 py-4 text-[12px] font-medium transition-colors',
                        active
                          ? 'border-[var(--lime)] bg-[var(--lime-soft)] text-[var(--lime)]'
                          : 'border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)] hover:bg-[var(--elevated-2)]',
                      )}
                    >
                      <Icon size={18} />
                      {opt.label}
                    </button>
                  )
                })}
              </div>
            </StepBlock>
          )}

          {step === 'deep' && (
            <StepBlock
              icon={<Eye size={18} />}
              title="Deep context"
              body={
                isMac()
                  ? 'When Accessibility allows, Kopiq can also remember window titles and, when possible, the page or file. History still works without it.'
                  : 'Kopiq can also remember window titles from the app you copied from. History still works without it.'
              }
            >
              <ToggleRow
                label="Save window titles and page or file when available"
                checked={deep}
                onChange={() => setDeep((v) => !v)}
              />
            </StepBlock>
          )}

          {step === 'sensitive' && (
            <StepBlock
              icon={<Shield size={18} />}
              title="Suspected secrets"
              body="API keys and similar strings can expire quickly, or never be saved."
            >
              <div className="flex gap-2">
                {(
                  [
                    { id: 'expire' as const, label: 'Expire after 60 seconds' },
                    { id: 'never' as const, label: 'Never save' },
                  ]
                ).map((opt) => (
                  <button
                    key={opt.id}
                    type="button"
                    onClick={() => setPolicy(opt.id)}
                    className={cn(
                      'flex-1 rounded-[10px] border px-3 py-2.5 text-left text-[12px] font-medium',
                      policy === opt.id
                        ? 'border-[var(--lime)] bg-[var(--lime-soft)] text-[var(--lime)]'
                        : 'border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)]',
                    )}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </StepBlock>
          )}

          {step === 'exclusions' && (
            <StepBlock
              icon={<Ban size={18} />}
              title="Excluded apps"
              body="Password managers are already excluded. You can add more anytime in Settings."
            >
              <ul className="overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--elevated)]">
                {exclusions.length === 0 ? (
                  <li className="px-3 py-2.5 text-[12px] text-[var(--subtle)]">
                    Builtin exclusions will appear after first launch.
                  </li>
                ) : (
                  exclusions.slice(0, 6).map((ex) => (
                    <li
                      key={ex.id}
                      className="flex items-center gap-2.5 border-b border-[var(--border)] px-3 py-2 text-[12px] text-[var(--mist)] last:border-b-0"
                    >
                      <AppIcon
                        name={ex.applicationName || ex.applicationId || 'App'}
                        iconKey={ex.iconCacheKey || ex.applicationId}
                        size={18}
                      />
                      <span className="min-w-0 truncate">
                        {ex.applicationName || ex.applicationId}
                      </span>
                    </li>
                  ))
                )}
              </ul>
              {frontmost && (frontmost.applicationId || frontmost.applicationName) && (
                <button
                  type="button"
                  disabled={excludedLaunch}
                  className="mt-3 w-full rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[12px] font-medium text-[var(--mist)] hover:bg-[var(--elevated-2)] disabled:opacity-50"
                  onClick={() =>
                    void excludeLaunchFrontmost().then(() => setExcludedLaunch(true))
                  }
                >
                  {excludedLaunch
                    ? `Excluded ${frontmost.applicationName || frontmost.applicationId}`
                    : `Exclude ${frontmost.applicationName || frontmost.applicationId}`}
                </button>
              )}
            </StepBlock>
          )}

          {step === 'permissions' && (
            <StepBlock
              icon={<KeyRound size={18} />}
              title="Accessibility"
              body="macOS needs Accessibility for window titles and automatic paste. Clipboard history still works without it — you paste manually."
            >
              <div className="flex items-center justify-between gap-3 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-3">
                <div>
                  <p className="text-[12px] font-medium text-[var(--mist)]">
                    {axOk ? 'Accessibility granted' : 'Accessibility not granted yet'}
                  </p>
                  <p className="text-[11px] text-[var(--subtle)]">
                    {axOk
                      ? 'Deep context and auto-paste can work.'
                      : 'Open System Settings, then enable Kopiq.'}
                  </p>
                </div>
                {axOk ? (
                  <Check size={16} className="shrink-0 text-[var(--lime)]" />
                ) : (
                  <button
                    type="button"
                    className="shrink-0 rounded-lg bg-[var(--lime)] px-3 py-1.5 text-[12px] font-semibold text-[var(--on-lime)]"
                    onClick={() => void openAccessibilitySettings().then(() => refreshAx())}
                  >
                    Open Settings
                  </button>
                )}
              </div>
              <p className="mt-2 text-[11px] text-[var(--subtle)]">
                You can continue either way. Grant later from Settings if you skip.
              </p>
            </StepBlock>
          )}

          {step === 'enable' && (
            <StepBlock
              icon={<Check size={18} />}
              title="Start capturing"
              body={`Kopiq will watch the clipboard from this moment on. Pause anytime from the ${trayLabel()}.`}
            />
          )}
        </div>

        {error && (
          <p className="mb-3 text-[12px] text-red-400" role="alert">
            {error}
          </p>
        )}

        <div className="flex items-center justify-between gap-3 border-t border-[var(--border)] pt-4">
          <button
            type="button"
            onClick={back}
            disabled={stepIdx === 0 || busy}
            className="rounded-lg px-3 py-2 text-[13px] text-[var(--muted)] hover:text-[var(--mist)] disabled:opacity-30"
          >
            Back
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => void next()}
            className="rounded-lg bg-[var(--lime)] px-4 py-2 text-[13px] font-semibold text-[var(--on-lime)] disabled:opacity-60"
          >
            {step === 'enable' ? (busy ? 'Starting…' : 'Enable capture') : 'Continue'}
          </button>
        </div>
      </div>
    </div>
  )
}

function StepBlock({
  icon,
  title,
  body,
  children,
}: {
  icon: ReactNode
  title: string
  body: string
  children?: ReactNode
}) {
  return (
    <div>
      <div className="mb-3 flex h-9 w-9 items-center justify-center rounded-[10px] bg-[var(--lime-soft)] text-[var(--lime)]">
        {icon}
      </div>
      <h2 className="text-[22px] font-semibold tracking-tight text-[var(--mist)]">{title}</h2>
      <p className="mt-2 text-[14px] leading-relaxed text-[var(--muted)]">{body}</p>
      {children && <div className="mt-5 space-y-3">{children}</div>}
    </div>
  )
}

function ToggleRow({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: () => void
}) {
  return (
    <div className="flex items-start justify-between gap-4 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-3">
      <p className="text-[12px] font-medium text-[var(--mist)]">{label}</p>
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
