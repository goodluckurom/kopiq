import { useEffect, useState, type ReactNode } from 'react'
import {
  Layers,
  Pin,
  Image as ImageIcon,
  Code2,
  Link2,
  FileText,
  File,
  Bookmark,
  FolderOpen,
  ListOrdered,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
} from 'lucide-react'
import type { CategoryId, HistoryCounts, NavFilter } from '../../lib/types'
import { fetchCounts } from '../../lib/api'
import { getStoredAppearance, resolveAppearance } from '../../lib/appearance'
import { cn, categoryTitle } from '../../lib/utils'
import { AppIcon } from '../shared/AppIcon'

const CATEGORIES: {
  id: CategoryId
  icon: typeof Layers
  dark: string
  light: string
  softDark: string
  softLight: string
}[] = [
  {
    id: 'all',
    icon: Layers,
    dark: '#a3ff12',
    light: '#4d8a00',
    softDark: 'rgb(163 255 18 / 0.15)',
    softLight: 'rgb(77 138 0 / 0.16)',
  },
  {
    id: 'pinned',
    icon: Pin,
    dark: '#f5c542',
    light: '#a67c00',
    softDark: 'rgb(245 197 66 / 0.15)',
    softLight: 'rgb(166 124 0 / 0.16)',
  },
  {
    id: 'images',
    icon: ImageIcon,
    dark: '#b794f6',
    light: '#6d28d9',
    softDark: 'rgb(183 148 246 / 0.15)',
    softLight: 'rgb(109 40 217 / 0.14)',
  },
  {
    id: 'code',
    icon: Code2,
    dark: '#4db8ff',
    light: '#0369a1',
    softDark: 'rgb(77 184 255 / 0.15)',
    softLight: 'rgb(3 105 161 / 0.14)',
  },
  {
    id: 'links',
    icon: Link2,
    dark: '#60a5fa',
    light: '#1d4ed8',
    softDark: 'rgb(96 165 250 / 0.15)',
    softLight: 'rgb(29 78 216 / 0.14)',
  },
  {
    id: 'text',
    icon: FileText,
    dark: '#94a3b8',
    light: '#475569',
    softDark: 'rgb(148 163 184 / 0.15)',
    softLight: 'rgb(71 85 105 / 0.14)',
  },
  {
    id: 'files',
    icon: File,
    dark: '#fb923c',
    light: '#c2410c',
    softDark: 'rgb(251 146 60 / 0.15)',
    softLight: 'rgb(194 65 12 / 0.14)',
  },
  {
    id: 'snippets',
    icon: Bookmark,
    dark: '#fb7185',
    light: '#be123c',
    softDark: 'rgb(251 113 133 / 0.15)',
    softLight: 'rgb(190 18 60 / 0.14)',
  },
  {
    id: 'collections',
    icon: FolderOpen,
    dark: '#2dd4bf',
    light: '#0f766e',
    softDark: 'rgb(45 212 191 / 0.15)',
    softLight: 'rgb(15 118 110 / 0.14)',
  },
  {
    id: 'stacks',
    icon: ListOrdered,
    dark: '#f472b6',
    light: '#be185d',
    softDark: 'rgb(244 114 182 / 0.15)',
    softLight: 'rgb(190 24 93 / 0.14)',
  },
]

type Props = {
  filter: NavFilter
  onFilterChange: (filter: NavFilter) => void
  onOpenSettings?: () => void
  settingsActive?: boolean
  refreshKey?: number
  collapsed?: boolean
  onToggleCollapsed?: () => void
  clearTitlebar?: boolean
}

const EMPTY_COUNTS: HistoryCounts = {
  total: 0,
  pinned: 0,
  byCategory: {},
  byApp: [],
}

export function Sidebar({
  filter,
  onFilterChange,
  onOpenSettings,
  settingsActive,
  refreshKey = 0,
  collapsed = false,
  onToggleCollapsed,
  clearTitlebar = false,
}: Props) {
  const [counts, setCounts] = useState<HistoryCounts>(EMPTY_COUNTS)
  const [theme, setTheme] = useState<'light' | 'dark'>(() =>
    resolveAppearance(getStoredAppearance()),
  )

  useEffect(() => {
    const sync = () => setTheme(resolveAppearance(getStoredAppearance()))
    sync()
    const obs = new MutationObserver(sync)
    obs.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-theme', 'style'],
    })
    const mq = window.matchMedia('(prefers-color-scheme: light)')
    mq.addEventListener('change', sync)
    return () => {
      obs.disconnect()
      mq.removeEventListener('change', sync)
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    const load = () => {
      void fetchCounts().then((c) => {
        if (!cancelled) setCounts(c)
      })
    }
    load()

    const onCounts = () => load()
    window.addEventListener('kopiq:counts-changed', onCounts)

    let off: (() => void) | undefined
    let alive = true
    void import('@wailsio/runtime')
      .then(({ Events }) => {
        if (!alive) return
        off = Events.On('history.changed', load)
      })
      .catch(() => {})

    return () => {
      cancelled = true
      alive = false
      window.removeEventListener('kopiq:counts-changed', onCounts)
      off?.()
    }
  }, [refreshKey])

  const categoryCount = (id: CategoryId) => {
    if (id === 'all') return counts.total
    if (id === 'pinned') return counts.pinned
    if (id === 'snippets') return counts.snippets ?? 0
    if (id === 'collections') return counts.collections ?? 0
    if (id === 'stacks') return counts.stacks ?? 0
    return counts.byCategory[id] ?? 0
  }

  return (
    <aside
      className={cn(
        'flex h-full shrink-0 flex-col border-r border-[var(--border)] bg-[var(--surface)] transition-[width] duration-150',
        collapsed ? 'w-[64px]' : 'w-[240px]',
      )}
    >
      <div
        className={cn(
          'flex items-center gap-2.5 pb-3',
          collapsed ? 'justify-center px-2' : 'px-4',
          !clearTitlebar && 'pt-4',
        )}
        style={clearTitlebar ? { paddingTop: 'var(--titlebar-pad-top)' } : undefined}
      >
        <img
          src="/brand/icon.png"
          alt=""
          className="h-7 w-7 rounded-md object-contain"
        />
        {!collapsed && (
          <span className="text-[15px] font-semibold tracking-tight text-[var(--mist)]">
            Kopiq
          </span>
        )}
      </div>

      <nav className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
        {!collapsed && <SectionLabel>Categories</SectionLabel>}
        <ul className="mb-4 flex flex-col gap-0.5">
          {CATEGORIES.map(({ id, icon: Icon, dark, light, softDark, softLight }) => {
            const active =
              filter.category === id && !filter.sourceApp && !filter.tag
            const count = categoryCount(id)
            const color = theme === 'light' ? light : dark
            const soft = theme === 'light' ? softLight : softDark
            return (
              <li key={id}>
                <button
                  type="button"
                  title={categoryTitle(id)}
                  onClick={() => onFilterChange({ category: id })}
                  className={cn(
                    'flex w-full items-center rounded-lg py-1.5 text-[13px] transition-colors duration-[120ms]',
                    collapsed ? 'justify-center px-0' : 'gap-2.5 px-2.5',
                    active
                      ? 'font-semibold text-[var(--on-lime)]'
                      : 'text-[var(--mist)] hover:bg-[var(--elevated)]',
                  )}
                  style={
                    active
                      ? { background: 'var(--lime)', color: 'var(--on-lime)' }
                      : undefined
                  }
                >
                  <span
                    className={cn(
                      'flex h-6 w-6 shrink-0 items-center justify-center rounded-md',
                      active && 'bg-[rgb(0_0_0_/0.12)]',
                    )}
                    style={
                      active
                        ? undefined
                        : { background: soft, color }
                    }
                  >
                    <Icon
                      size={14}
                      strokeWidth={active ? 2.25 : 2}
                      style={active ? undefined : { color }}
                    />
                  </span>
                  {!collapsed && (
                    <>
                      <span className="flex-1 truncate text-left">
                        {categoryTitle(id)}
                      </span>
                      <span
                        className={cn(
                          'tabular-nums text-[11px]',
                          active ? 'text-[var(--on-lime)]/70' : 'text-[var(--subtle)]',
                        )}
                      >
                        {count}
                      </span>
                    </>
                  )}
                </button>
              </li>
            )
          })}
        </ul>

        {counts.byApp.length > 0 && (
          <>
            {!collapsed && <SectionLabel>Sources</SectionLabel>}
            <ul className={cn('mb-4 flex flex-col gap-0.5', collapsed && 'mt-2')}>
              {counts.byApp.map((src) => {
                const active = filter.sourceApp === src.name
                return (
                  <li key={`${src.appId}-${src.name}`}>
                    <button
                      type="button"
                      title={src.name}
                      onClick={() =>
                        onFilterChange({
                          category: 'all',
                          sourceApp: active ? undefined : src.name,
                        })
                      }
                      className={cn(
                        'flex w-full items-center rounded-lg text-[13px] transition-colors duration-[120ms]',
                        collapsed
                          ? 'justify-center px-0 py-1.5'
                          : 'gap-2.5 px-2.5 py-2',
                        active
                          ? 'bg-[var(--elevated)] text-[var(--mist)] ring-1 ring-[var(--lime)]/40'
                          : 'text-[var(--mist)] hover:bg-[var(--elevated)]',
                      )}
                    >
                      <AppIcon
                        name={src.name}
                        iconKey={src.iconCacheKey || src.appId}
                        size={collapsed ? 24 : 22}
                      />
                      {!collapsed && (
                        <>
                          <span className="flex-1 truncate text-left">{src.name}</span>
                          <span className="tabular-nums text-[11px] text-[var(--subtle)]">
                            {src.count}
                          </span>
                        </>
                      )}
                    </button>
                  </li>
                )
              })}
            </ul>
          </>
        )}
      </nav>

      <div
        className={cn(
          'shrink-0 border-t border-[var(--border)] bg-[var(--elevated)]/40',
          collapsed ? 'px-1.5 py-2' : 'px-2.5 py-2.5',
        )}
      >
        <div
          className={cn(
            'flex rounded-[10px] border border-[var(--border)] bg-[var(--surface)] p-1',
            collapsed ? 'flex-col gap-1' : 'items-center gap-1',
          )}
        >
          <button
            type="button"
            className={cn(
              'inline-flex items-center gap-2 rounded-[8px] px-2.5 py-2 text-[12px] font-medium transition-colors',
              collapsed ? 'justify-center' : 'flex-1',
              settingsActive
                ? 'bg-[var(--lime-soft)] text-[var(--lime)]'
                : 'text-[var(--muted)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]',
            )}
            aria-label="Settings"
            onClick={() => onOpenSettings?.()}
          >
            <Settings size={15} strokeWidth={1.85} />
            {!collapsed && <span>Settings</span>}
          </button>
          <button
            type="button"
            className={cn(
              'inline-flex items-center justify-center rounded-[8px] px-2.5 py-2 text-[var(--muted)] transition-colors hover:bg-[var(--elevated)] hover:text-[var(--mist)]',
              collapsed ? '' : 'shrink-0',
            )}
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            aria-pressed={collapsed}
            title={collapsed ? 'Expand' : 'Collapse'}
            onClick={() => onToggleCollapsed?.()}
          >
            {collapsed ? <PanelLeftOpen size={15} /> : <PanelLeftClose size={15} />}
          </button>
        </div>
      </div>
    </aside>
  )
}

function SectionLabel({ children }: { children: ReactNode }) {
  return (
    <p className="mb-1 px-2.5 pt-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--subtle)]">
      {children}
    </p>
  )
}
