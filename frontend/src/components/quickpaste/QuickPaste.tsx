import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Copy, Sparkles } from 'lucide-react'
import {
  copyItemOnly,
  dismissQuickPaste,
  getSettings,
  listTransformActions,
  needsOnboarding,
  pasteItem,
  pasteTransformed,
  stackStatus,
  stopStack,
  type TransformAction,
} from '../../lib/api'
import { syncAppearanceFromSettings } from '../../lib/appearance'
import { modGlyph, trayLabel } from '../../lib/platform'
import { useClipboardHistory } from '../../lib/useClipboardHistory'
import { ItemCard } from '../library/ItemCard'
import { SearchBar } from '../library/SearchBar'
import { VirtualList } from '../shared/VirtualList'
import { cn } from '../../lib/utils'

export function QuickPaste() {
  const [query, setQuery] = useState('')
  const { items, loading } = useClipboardHistory(query)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [toast, setToast] = useState<string | null>(null)
  const [setupNeeded, setSetupNeeded] = useState(false)
  const [convertOpen, setConvertOpen] = useState(false)
  const [convertActions, setConvertActions] = useState<TransformAction[]>([])
  const [convertIndex, setConvertIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    void syncAppearanceFromSettings(getSettings)
  }, [])

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    void needsOnboarding().then(setSetupNeeded)
  }, [])

  useEffect(() => {
    let cancelled = false
    const tick = () => {
      void stackStatus().then((st) => {
        if (!cancelled && st.armed) setToast(st.label)
      })
    }
    tick()
    const id = window.setInterval(tick, 2000)
    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [])

  useEffect(() => {
    let offOpened: (() => void) | undefined
    let offAppearance: (() => void) | undefined
    void (async () => {
      try {
        const { Events } = await import('@wailsio/runtime')
        offOpened = Events.On('quickpaste.opened', () => {
          void syncAppearanceFromSettings(getSettings)
          setQuery('')
          setToast(null)
          setSelectedIndex(0)
          setConvertOpen(false)
          setConvertActions([])
          window.setTimeout(() => inputRef.current?.focus(), 30)
        })
        offAppearance = Events.On('appearance.changed', () => {
          void syncAppearanceFromSettings(getSettings)
        })
      } catch {
        // not in runtime
      }
    })()
    return () => {
      if (typeof offOpened === 'function') offOpened()
      if (typeof offAppearance === 'function') offAppearance()
    }
  }, [])

  useEffect(() => {
    setSelectedIndex(0)
    setConvertOpen(false)
  }, [items])

  useEffect(() => {
    if (!convertOpen) return
    const el = listRef.current?.querySelector<HTMLElement>(
      `[data-qp-index="${convertIndex}"]`,
    )
    el?.scrollIntoView({ block: 'nearest' })
  }, [convertOpen, convertIndex])

  const selected = items[selectedIndex] ?? null

  const showToast = useCallback((message: string, ms = 1400) => {
    setToast(message)
    window.setTimeout(() => setToast(null), ms)
  }, [])

  const doPaste = useCallback(
    async (id: number) => {
      const result = await pasteItem(id)
      if (result.message) showToast(result.message, 2200)
    },
    [showToast],
  )

  const doCopy = useCallback(
    async (id: number) => {
      try {
        await copyItemOnly(id)
        showToast('Copied')
      } catch {
        showToast('Copy failed')
      }
    },
    [showToast],
  )

  const openConvert = useCallback(
    async (itemId?: number) => {
      const id = itemId ?? selected?.id
      if (id == null) return
      const actions = await listTransformActions(id)
      if (actions.length === 0) {
        showToast('No transforms for this item')
        return
      }
      if (itemId != null) {
        const idx = items.findIndex((it) => it.id === itemId)
        if (idx >= 0) setSelectedIndex(idx)
      }
      setConvertActions(actions)
      setConvertIndex(0)
      setConvertOpen(true)
    },
    [selected, showToast, items],
  )

  const doConvertPaste = useCallback(
    async (transformId: string) => {
      if (!selected) return
      try {
        const result = await pasteTransformed(selected.id, transformId)
        setConvertOpen(false)
        if (result.message) showToast(result.message, 2200)
        else if (result.pasted) showToast('Pasted')
        else showToast('Copied')
      } catch (e) {
        showToast(e instanceof Error ? e.message : 'Transform failed', 2200)
      }
    },
    [selected, showToast],
  )

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (convertOpen) {
        if (e.key === 'ArrowDown') {
          e.preventDefault()
          setConvertIndex((i) => Math.min(i + 1, Math.max(0, convertActions.length - 1)))
          return
        }
        if (e.key === 'ArrowUp') {
          e.preventDefault()
          setConvertIndex((i) => Math.max(i - 1, 0))
          return
        }
        if (e.key === 'Enter') {
          e.preventDefault()
          const action = convertActions[convertIndex]
          if (action) void doConvertPaste(action.id)
          return
        }
        if (e.key === 'Escape' || e.key === 'Tab' || e.key === '\\') {
          e.preventDefault()
          setConvertOpen(false)
          return
        }
        return
      }

      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((i) => Math.min(i + 1, Math.max(0, items.length - 1)))
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((i) => Math.max(i - 1, 0))
        return
      }
      if (e.key === 'Enter') {
        e.preventDefault()
        if (selected) void doPaste(selected.id)
        return
      }
      if (e.key === 'Tab' || (e.key === '\\' && !e.metaKey && !e.ctrlKey)) {
        e.preventDefault()
        void openConvert()
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        void stopStack()
        void dismissQuickPaste()
        return
      }
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'c') {
        const input = inputRef.current
        if (
          input &&
          document.activeElement === input &&
          input.selectionStart !== input.selectionEnd
        ) {
          return
        }
        if (!selected) return
        e.preventDefault()
        void doCopy(selected.id)
        return
      }
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'o') {
        e.preventDefault()
        window.location.hash = '#/'
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [
    items.length,
    selected,
    doPaste,
    doCopy,
    convertOpen,
    convertActions,
    convertIndex,
    doConvertPaste,
    openConvert,
  ])

  const copyHint = modGlyph() === '⌘' ? '⌘C' : 'Ctrl+C'
  const hint = useMemo(
    () =>
      convertOpen
        ? [
            { keys: '↑↓', label: 'convert' },
            { keys: '↵', label: 'paste result' },
            { keys: 'esc', label: 'back' },
          ]
        : [
            { keys: '↑↓', label: 'navigate' },
            { keys: '↵', label: 'paste' },
            { keys: '⇥', label: 'convert' },
            { keys: copyHint, label: 'copy' },
            { keys: 'esc', label: 'dismiss' },
          ],
    [convertOpen, copyHint],
  )

  return (
    <div
      className={cn(
        'relative flex h-full w-full flex-col overflow-hidden',
        'bg-[var(--surface)] animate-fade-in',
      )}
    >
        <div className="qp-drag shrink-0 border-b border-[var(--border)] p-3">
          <SearchBar
            value={query}
            onChange={setQuery}
            placeholder="Search…  :alias  collection:name  type:json"
            showShortcut={false}
            autoFocus
            inputRef={inputRef}
            className="qp-no-drag"
          />
        </div>

        {convertOpen ? (
          <div ref={listRef} className="min-h-0 flex-1 overflow-y-auto p-2">
            <div className="flex flex-col gap-0.5">
              <p className="px-2 py-1.5 text-[11px] font-medium text-[var(--subtle)]">
                Convert · {selected?.title || selected?.textPreview?.slice(0, 40) || 'item'}
              </p>
              {convertActions.map((action, index) => {
                const isSelected = index === convertIndex
                return (
                  <button
                    key={action.id}
                    type="button"
                    data-qp-index={index}
                    onClick={() => setConvertIndex(index)}
                    onDoubleClick={() => void doConvertPaste(action.id)}
                    className={cn(
                      'flex items-center gap-2 rounded-[10px] px-3 py-2.5 text-left transition-colors',
                      isSelected
                        ? 'bg-[var(--lime-soft)] text-[var(--mist)]'
                        : 'text-[var(--muted)] hover:bg-[var(--elevated)]',
                    )}
                  >
                    <Sparkles size={14} className="shrink-0 text-[var(--lime)]" />
                    <span className="min-w-0 flex-1 text-[13px] font-medium">{action.label}</span>
                    <span className="text-[10px] uppercase tracking-wide text-[var(--subtle)]">
                      {action.group}
                    </span>
                  </button>
                )
              })}
            </div>
          </div>
        ) : setupNeeded && items.length === 0 ? (
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            <div className="px-3 py-10 text-center">
              <p className="text-[13px] text-[var(--mist)]">
                Finish setup to start capturing.
              </p>
              <p className="mt-1 text-[12px] text-[var(--subtle)]">
                Open the library from the {trayLabel()} to continue onboarding.
              </p>
            </div>
          </div>
        ) : loading && items.length === 0 ? (
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            <p className="px-3 py-10 text-center text-[13px] text-[var(--subtle)]">
              Loading…
            </p>
          </div>
        ) : items.length === 0 ? (
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            <p className="px-3 py-10 text-center text-[13px] text-[var(--subtle)]">
              {query
                ? 'Nothing copied matches that.'
                : 'Copies you make will show up here.'}
            </p>
          </div>
        ) : (
          <VirtualList
            scrollParentRef={listRef}
            count={items.length}
            estimateSize={() => 64}
            overscan={12}
            scrollToIndex={selectedIndex}
            scrollAlign="auto"
            className="p-2"
            aria-label="Quick Paste"
          >
            {(index) => {
              const item = items[index]!
              const isSelected = index === selectedIndex
              return (
                <div className="group relative mb-0.5 flex items-stretch gap-1">
                  <div className="min-w-0 flex-1">
                    <ItemCard
                      item={item}
                      compact
                      selected={isSelected}
                      onSelect={() => setSelectedIndex(index)}
                      onDoubleClick={() => void doPaste(item.id)}
                    />
                  </div>
                  <button
                    type="button"
                    title="Convert (Tab)"
                    aria-label="Convert"
                    className={cn(
                      'my-1 flex w-8 shrink-0 items-center justify-center rounded-lg border border-transparent text-[var(--subtle)] transition-all duration-[120ms]',
                      'hover:border-[var(--border)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]',
                      isSelected
                        ? 'opacity-100'
                        : 'opacity-0 group-hover:opacity-100 focus-visible:opacity-100',
                    )}
                    onClick={(e) => {
                      e.stopPropagation()
                      void openConvert(item.id)
                    }}
                  >
                    <Sparkles size={14} />
                  </button>
                  <button
                    type="button"
                    title={`Copy (${copyHint})`}
                    aria-label="Copy to clipboard"
                    className={cn(
                      'my-1 mr-1 flex w-8 shrink-0 items-center justify-center rounded-lg border border-transparent text-[var(--subtle)] transition-all duration-[120ms]',
                      'hover:border-[var(--border)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]',
                      isSelected
                        ? 'opacity-100'
                        : 'opacity-0 group-hover:opacity-100 focus-visible:opacity-100',
                    )}
                    onClick={(e) => {
                      e.stopPropagation()
                      setSelectedIndex(index)
                      void doCopy(item.id)
                    }}
                  >
                    <Copy size={14} />
                  </button>
                </div>
              )
            }}
          </VirtualList>
        )}

        <div className="flex shrink-0 items-center justify-between gap-3 border-t border-[var(--border)] px-3 py-2">
          <div className="flex flex-wrap gap-x-3 gap-y-1">
            {hint.map((h) => (
              <span key={h.keys} className="text-[10px] text-[var(--subtle)]">
                <kbd className="rounded border border-[var(--border)] bg-[var(--elevated)] px-1 font-mono text-[10px] text-[var(--muted)]">
                  {h.keys}
                </kbd>{' '}
                {h.label}
              </span>
            ))}
          </div>
          {toast && (
            <span className="text-[11px] font-medium text-[var(--lime)] animate-fade-in">
              {toast}
            </span>
          )}
        </div>
    </div>
  )
}
