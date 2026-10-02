import { useEffect, useRef, useState } from 'react'
import type { ClipboardItem, NavFilter } from '../../lib/types'
import {
  clearHistory,
  copyItem,
  deleteItem,
  getItem,
  getSettings,
  needsOnboarding,
  pasteItem,
  pinItem,
  openFile,
  revealFile,
  revealImage,
  setSetting,
} from '../../lib/api'
import { syncAppearanceFromSettings } from '../../lib/appearance'
import { isFileItem, matchesFilter } from '../../lib/utils'
import { useClipboardHistory } from '../../lib/useClipboardHistory'
import { Sidebar } from './Sidebar'
import { ItemList } from './ItemList'
import { DetailPane } from './DetailPane'
import { RowActionMenu, type RowAction } from './RowActionMenu'
import {
  OrganizePane,
  promptSaveSnippet,
} from './OrganizePane'
import { SettingsPane } from '../settings/SettingsPane'
import { OnboardingPane } from '../onboarding/OnboardingPane'
import { ConfirmModal } from '../shared/ConfirmModal'
import { Settings, PanelRightClose } from 'lucide-react'

type ConfirmKind = null | 'delete' | 'clear'

export function Library() {
  const [filter, setFilter] = useState<NavFilter>({ category: 'all' })
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const { items, setItems, loading, reload } = useClipboardHistory(debouncedQuery)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [checkedIds, setCheckedIds] = useState<number[]>([])
  const [toast, setToast] = useState<string | null>(null)
  const [detail, setDetail] = useState<ClipboardItem | null>(null)
  const [detailOpen, setDetailOpen] = useState(true)
  const [showSettings, setShowSettings] = useState(false)
  const [showOnboarding, setShowOnboarding] = useState(false)
  const [onboardingChecked, setOnboardingChecked] = useState(false)
  const [confirm, setConfirm] = useState<ConfirmKind>(null)
  const [confirmBusy, setConfirmBusy] = useState(false)
  const [menuItem, setMenuItem] = useState<ClipboardItem | null>(null)
  const [menuAnchor, setMenuAnchor] = useState<{ top: number; left: number } | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    try {
      return window.localStorage.getItem('kopiq.sidebar.collapsed') === '1'
    } catch {
      return false
    }
  })
  const searchRef = useRef<HTMLInputElement>(null)
  const prevTopId = useRef<number | null>(null)

  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedQuery(query), 120)
    return () => window.clearTimeout(t)
  }, [query])

  const toggleSidebar = () => {
    setSidebarCollapsed((prev) => {
      const next = !prev
      try {
        window.localStorage.setItem('kopiq.sidebar.collapsed', next ? '1' : '0')
      } catch {
        // ignore
      }
      return next
    })
  }

  useEffect(() => {
    void syncAppearanceFromSettings(getSettings)
    let off: (() => void) | undefined
    void (async () => {
      try {
        const { Events } = await import('@wailsio/runtime')
        off = Events.On('appearance.changed', () => {
          void syncAppearanceFromSettings(getSettings)
        })
      } catch {
        // not in runtime
      }
    })()
    return () => {
      if (typeof off === 'function') off()
    }
  }, [])

  useEffect(() => {
    const openSettings = () => setShowSettings(true)
    const openOnboarding = () => setShowOnboarding(true)
    const openPinned = () => {
      setShowSettings(false)
      setShowOnboarding(false)
      setFilter({ category: 'pinned' })
    }
    window.addEventListener('kopiq:open-settings', openSettings)
    window.addEventListener('kopiq:open-onboarding', openOnboarding)
    window.addEventListener('kopiq:open-pinned', openPinned)
    const onHash = () => {
      if (window.location.hash.includes('settings')) setShowSettings(true)
      if (window.location.hash.includes('onboarding')) setShowOnboarding(true)
    }
    onHash()
    window.addEventListener('hashchange', onHash)
    void needsOnboarding().then(async (needs) => {
      if (needs) {
        if (import.meta.env.DEV) {
          try {
            await setSetting('capture.enabled', 'false')
          } catch {
            // ignore
          }
        }
        setShowOnboarding(true)
      }
      setOnboardingChecked(true)
    })
    return () => {
      window.removeEventListener('kopiq:open-settings', openSettings)
      window.removeEventListener('kopiq:open-onboarding', openOnboarding)
      window.removeEventListener('kopiq:open-pinned', openPinned)
      window.removeEventListener('hashchange', onHash)
    }
  }, [])

  useEffect(() => {
    if (!items.length) {
      setSelectedId(null)
      prevTopId.current = null
      return
    }
    const top = items[0]?.id ?? null
    if (
      selectedId == null ||
      (top != null && top !== prevTopId.current && selectedId === prevTopId.current)
    ) {
      setSelectedId(top)
    } else if (selectedId != null && !items.some((i) => i.id === selectedId)) {
      setSelectedId(top)
    }
    prevTopId.current = top
  }, [items, selectedId])

  useEffect(() => {
    if (selectedId == null) {
      setDetail(null)
      return
    }
    const summary = items.find((i) => i.id === selectedId) ?? null
    setDetail(summary)
    let cancelled = false
    void getItem(selectedId).then((full) => {
      if (!cancelled && full) setDetail(full)
    })
    return () => {
      cancelled = true
    }
  }, [selectedId, items])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (showOnboarding || showSettings) return
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        searchRef.current?.focus()
      }
      if ((e.metaKey || e.ctrlKey) && e.key === ',') {
        e.preventDefault()
        setShowSettings(true)
      }
      const target = e.target as HTMLElement | null
      const typing =
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.isContentEditable)
      if (
        !typing &&
        selectedId != null &&
        (e.key === 'Delete' || e.key === 'Backspace') &&
        !e.metaKey &&
        !e.ctrlKey &&
        !e.altKey
      ) {
        e.preventDefault()
        setConfirm('delete')
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [showOnboarding, showSettings, selectedId])

  const filtered = items.filter((item) => matchesFilter(item, filter))

  const removeFromState = (id: number) => {
    setItems((prev) => prev.filter((i) => i.id !== id))
    if (selectedId === id) {
      setSelectedId(null)
      setDetail(null)
    }
  }

  const runConfirm = async () => {
    if (!confirm) return
    setConfirmBusy(true)
    try {
      if (confirm === 'delete' && selectedId != null) {
        await deleteItem(selectedId)
        removeFromState(selectedId)
        void reload()
      } else if (confirm === 'clear') {
        await clearHistory()
        setItems((prev) => prev.filter((i) => i.isPinned))
        void reload()
      }
      setConfirm(null)
    } catch {
      // keep modal open
    } finally {
      setConfirmBusy(false)
    }
  }

  const togglePin = async (item?: ClipboardItem | null) => {
    const target = item ?? detail
    if (!target) return
    if (target.sensitivity === 'suspected' || target.sensitivity === 'secret') return
    const next = !target.isPinned
    setItems((prev) =>
      prev.map((i) => (i.id === target.id ? { ...i, isPinned: next } : i)),
    )
    if (detail?.id === target.id) setDetail({ ...detail, isPinned: next })
    try {
      await pinItem(target.id, next)
      void reload()
    } catch {
      setItems((prev) =>
        prev.map((i) => (i.id === target.id ? { ...i, isPinned: !next } : i)),
      )
      if (detail?.id === target.id) setDetail({ ...detail, isPinned: !next })
    }
  }

  const handleRowAction = async (action: RowAction) => {
    if (!menuItem) return
    const item = menuItem
    switch (action) {
      case 'copy':
        await copyItem(item.id)
        break
      case 'paste':
        await pasteItem(item.id)
        break
      case 'pin':
      case 'unpin':
        await togglePin(item)
        break
      case 'open':
        try {
          await openFile(item.id)
        } catch {
          // ignore
        }
        break
      case 'reveal':
        try {
          if (isFileItem(item)) await revealFile(item.id)
          else await revealImage(item.id)
        } catch {
          // ignore
        }
        break
      case 'delete':
        setSelectedId(item.id)
        setConfirm('delete')
        break
      case 'filter-app':
        if (item.context?.applicationName) {
          setFilter({
            category: 'all',
            sourceApp: item.context.applicationName,
          })
        }
        break
      case 'save-snippet':
        try {
          await promptSaveSnippet(item)
          void reload()
        } catch {
          // ignore
        }
        break
    }
  }

  if (!onboardingChecked) {
    return (
      <div className="flex h-full items-center justify-center bg-[var(--charcoal)] text-[13px] text-[var(--subtle)]">
        Loading…
      </div>
    )
  }

  if (showOnboarding) {
    return (
      <OnboardingPane
        onDone={() => {
          setShowOnboarding(false)
          if (window.location.hash.includes('onboarding')) {
            window.location.hash = '#/'
          }
          void reload()
        }}
      />
    )
  }

  if (showSettings) {
    return (
      <div className="flex h-full bg-[var(--charcoal)]">
        <Sidebar
          filter={filter}
          onFilterChange={(f) => {
            setFilter(f)
            setShowSettings(false)
          }}
          onOpenSettings={() => setShowSettings(true)}
          settingsActive
          refreshKey={items.length + (items[0]?.lastCopiedAt ?? 0)}
          collapsed={sidebarCollapsed}
          onToggleCollapsed={toggleSidebar}
          clearTitlebar
        />
        <SettingsPane
          onBack={() => {
            setShowSettings(false)
            if (window.location.hash.includes('settings')) {
              window.location.hash = '#/'
            }
          }}
        />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col bg-[var(--charcoal)]">
      <ConfirmModal
        open={confirm === 'delete'}
        title="Remove this item?"
        description="It will be deleted from your clipboard history. This cannot be undone."
        confirmLabel="Remove"
        danger
        busy={confirmBusy}
        onCancel={() => {
          if (!confirmBusy) setConfirm(null)
        }}
        onConfirm={() => void runConfirm()}
      />
      <ConfirmModal
        open={confirm === 'clear'}
        title="Clear clipboard history?"
        description="All unpinned items will be removed. Pinned items are kept."
        confirmLabel="Clear all"
        danger
        busy={confirmBusy}
        onCancel={() => {
          if (!confirmBusy) setConfirm(null)
        }}
        onConfirm={() => void runConfirm()}
      />
      {menuItem && (
        <RowActionMenu
          item={menuItem}
          open={menuAnchor != null}
          anchor={menuAnchor}
          onClose={() => {
            setMenuAnchor(null)
            setMenuItem(null)
          }}
          onAction={(a) => void handleRowAction(a)}
          onCollectionAdded={(name) => {
            setToast(`Added to ${name}`)
            window.setTimeout(() => setToast(null), 2200)
            void reload()
          }}
        />
      )}

      <header
        className="flex h-10 shrink-0 items-center justify-end gap-1 border-b border-[var(--border)] pr-3"
        style={{ paddingLeft: 'var(--titlebar-pad-left)' }}
      >
        <button
          type="button"
          className="rounded-lg p-1.5 text-[var(--subtle)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]"
          aria-label="Settings"
          onClick={() => setShowSettings(true)}
        >
          <Settings size={15} />
        </button>
        <button
          type="button"
          className="rounded-lg p-1.5 text-[var(--subtle)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]"
          aria-label="Toggle detail"
          onClick={() => setDetailOpen((v) => !v)}
        >
          <PanelRightClose size={15} />
        </button>
      </header>

      <div className="flex min-h-0 flex-1">
        <Sidebar
          filter={filter}
          onFilterChange={setFilter}
          onOpenSettings={() => setShowSettings(true)}
          refreshKey={items.length + (items[0]?.lastCopiedAt ?? 0)}
          collapsed={sidebarCollapsed}
          onToggleCollapsed={toggleSidebar}
        />
        <main className="min-w-0 flex-1 bg-[var(--charcoal)]">
          {filter.category === 'snippets' ||
          filter.category === 'collections' ||
          filter.category === 'stacks' ? (
            <OrganizePane
              filter={filter}
              onFilterChange={setFilter}
              selectedId={selectedId}
              onSelect={(id) => {
                setSelectedId(id)
                setDetailOpen(true)
              }}
              onOpenMenu={(item, anchor) => {
                setSelectedId(item.id)
                setMenuItem(item)
                setMenuAnchor(anchor)
              }}
              refreshKey={items.length + (items[0]?.lastCopiedAt ?? 0)}
            />
          ) : loading && items.length === 0 ? (
            <div className="flex h-full items-center justify-center text-[13px] text-[var(--subtle)]">
              Loading clipboard…
            </div>
          ) : (
            <ItemList
              items={filtered}
              allItems={items}
              selectedId={selectedId}
              checkedIds={checkedIds}
              query={query}
              filter={filter}
              onQueryChange={setQuery}
              onFilterChange={setFilter}
              onSelect={(id) => {
                setSelectedId(id)
                setDetailOpen(true)
              }}
              onCheckedChange={setCheckedIds}
              onOpenMenu={(item, anchor) => {
                setSelectedId(item.id)
                setMenuItem(item)
                setMenuAnchor(anchor)
              }}
              onRequestDelete={() => setConfirm('delete')}
              onRequestClearAll={() => setConfirm('clear')}
              searchRef={searchRef}
              onToast={(msg) => {
                setToast(msg)
                window.setTimeout(() => setToast(null), 2200)
              }}
            />
          )}
        </main>
        {toast && (
          <div className="pointer-events-none fixed bottom-4 left-1/2 z-50 -translate-x-1/2 rounded-[10px] border border-[var(--border)] bg-[var(--surface)] px-3 py-2 text-[12px] font-medium text-[var(--mist)] shadow-lg">
            {toast}
          </div>
        )}
        {detailOpen && (
          <DetailPane
            item={detail}
            onClose={() => setDetailOpen(false)}
            onTogglePin={() => void togglePin()}
            onDeleted={(id) => {
              removeFromState(id)
              void reload()
            }}
          />
        )}
      </div>
    </div>
  )
}
