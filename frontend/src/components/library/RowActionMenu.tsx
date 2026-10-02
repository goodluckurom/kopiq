import { useEffect, useRef, useState } from 'react'
import {
  Copy,
  Pin,
  PinOff,
  Trash2,
  FolderOpen,
  ExternalLink,
  AppWindow,
  ClipboardPaste,
  BookmarkPlus,
  FolderPlus,
  Plus,
  ChevronRight,
} from 'lucide-react'
import type { ClipboardItem, Collection } from '../../lib/types'
import {
  addToCollection,
  createCollection,
  listCollections,
} from '../../lib/api'
import { cn, isFileItem } from '../../lib/utils'

export type RowAction =
  | 'copy'
  | 'paste'
  | 'pin'
  | 'unpin'
  | 'reveal'
  | 'open'
  | 'delete'
  | 'filter-app'
  | 'save-snippet'
  | 'add-collection'

type Props = {
  item: ClipboardItem
  open: boolean
  anchor: { top: number; left: number } | null
  onClose: () => void
  onAction: (action: RowAction) => void
  onCollectionAdded?: (name: string) => void
}

export function RowActionMenu({
  item,
  open,
  anchor,
  onClose,
  onAction,
  onCollectionAdded,
}: Props) {
  const ref = useRef<HTMLDivElement>(null)
  const subRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const [collectionsOpen, setCollectionsOpen] = useState(false)
  const [collections, setCollections] = useState<Collection[]>([])
  const [loadingCols, setLoadingCols] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isImage =
    item.kind === 'image' ||
    item.detectedType === 'image' ||
    item.detectedType === 'screenshot'
  const isFile = isFileItem(item)
  const canSnippet =
    !isImage && Boolean((item.textContent || item.textPreview || '').trim())
  const canPin = item.sensitivity !== 'suspected' && item.sensitivity !== 'secret'
  const app = item.context?.applicationName

  useEffect(() => {
    if (!open) {
      setCollectionsOpen(false)
      setCreating(false)
      setNewName('')
      setError(null)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        if (creating) {
          setCreating(false)
          setNewName('')
          return
        }
        if (collectionsOpen) {
          setCollectionsOpen(false)
          return
        }
        onClose()
      }
    }
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node
      if (ref.current?.contains(t) || subRef.current?.contains(t)) return
      onClose()
    }
    window.addEventListener('keydown', onKey)
    window.addEventListener('mousedown', onDown)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('mousedown', onDown)
    }
  }, [open, onClose, collectionsOpen, creating])

  useEffect(() => {
    if (!collectionsOpen) return
    setLoadingCols(true)
    setError(null)
    void listCollections()
      .then((cols) => {
        setCollections(cols)
        if (cols.length === 0) setCreating(true)
      })
      .catch(() => setCollections([]))
      .finally(() => setLoadingCols(false))
  }, [collectionsOpen])

  useEffect(() => {
    if (creating) window.setTimeout(() => inputRef.current?.focus(), 20)
  }, [creating])

  if (!open || !anchor) return null

  const menuTop = Math.min(anchor.top, window.innerHeight - 320)
  const menuLeft = Math.min(anchor.left, window.innerWidth - 220)
  const subLeft = Math.min(menuLeft + 208, window.innerWidth - 220)
  const subTop = menuTop

  const items: {
    id: RowAction
    label: string
    icon: typeof Copy
    danger?: boolean
    hidden?: boolean
    submenu?: boolean
  }[] = [
    { id: 'copy', label: 'Copy', icon: Copy },
    { id: 'paste', label: 'Paste', icon: ClipboardPaste },
    {
      id: 'open',
      label: 'Open',
      icon: ExternalLink,
      hidden: !isFile,
    },
    {
      id: item.isPinned ? 'unpin' : 'pin',
      label: item.isPinned ? 'Unpin' : 'Pin',
      icon: item.isPinned ? PinOff : Pin,
      hidden: !canPin && !item.isPinned,
    },
    {
      id: 'save-snippet',
      label: 'Save snippet…',
      icon: BookmarkPlus,
      hidden: !canSnippet,
    },
    {
      id: 'add-collection',
      label: 'Add to collection',
      icon: FolderPlus,
      submenu: true,
    },
    {
      id: 'reveal',
      label: 'Reveal in Finder',
      icon: FolderOpen,
      hidden: !(isImage && item.imageKept !== false) && !isFile,
    },
    {
      id: 'filter-app',
      label: app ? `Show all from ${app}` : 'Show from this app',
      icon: AppWindow,
      hidden: !app,
    },
    { id: 'delete', label: 'Delete', icon: Trash2, danger: true },
  ]

  const pickCollection = async (id: number, name: string) => {
    setBusy(true)
    setError(null)
    try {
      await addToCollection(id, item.id)
      onCollectionAdded?.(name)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not add')
    } finally {
      setBusy(false)
    }
  }

  const createAndAdd = async () => {
    const name = newName.trim()
    if (!name) {
      setError('Enter a name')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const c = await createCollection(name)
      await addToCollection(c.id, item.id)
      onCollectionAdded?.(c.name)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not create')
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <div
        ref={ref}
        role="menu"
        className="fixed z-[70] min-w-[200px] overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--surface)] py-1 shadow-[0_12px_40px_rgb(0_0_0_/0.35)] animate-fade-in"
        style={{ top: menuTop, left: menuLeft }}
      >
        {items
          .filter((i) => !i.hidden)
          .map((i) => {
            const Icon = i.icon
            const active = i.submenu && collectionsOpen
            return (
              <button
                key={i.id}
                type="button"
                role="menuitem"
                className={cn(
                  'flex w-full items-center gap-2.5 px-3 py-2 text-left text-[12px] transition-colors',
                  i.danger
                    ? 'text-[rgb(240_120_120)] hover:bg-[rgb(220_70_70_/0.12)]'
                    : active
                      ? 'bg-[var(--elevated)] text-[var(--mist)]'
                      : 'text-[var(--mist)] hover:bg-[var(--elevated)]',
                )}
                onClick={() => {
                  if (i.submenu) {
                    setCollectionsOpen(true)
                    setCreating(false)
                    setNewName('')
                    setError(null)
                    return
                  }
                  onAction(i.id)
                  onClose()
                }}
              >
                <Icon size={14} strokeWidth={1.75} />
                <span className="flex-1">{i.label}</span>
                {i.submenu && (
                  <ChevronRight size={14} className="text-[var(--subtle)]" />
                )}
              </button>
            )
          })}
      </div>

      {collectionsOpen && (
        <div
          ref={subRef}
          role="menu"
          aria-label="Collections"
          className="fixed z-[71] min-w-[200px] max-w-[260px] overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--surface)] py-1 shadow-[0_12px_40px_rgb(0_0_0_/0.35)] animate-fade-in"
          style={{ top: subTop, left: subLeft }}
        >
          {loadingCols ? (
            <p className="px-3 py-2 text-[12px] text-[var(--subtle)]">Loading…</p>
          ) : (
            <>
              {collections.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  role="menuitem"
                  disabled={busy}
                  className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-[12px] text-[var(--mist)] transition-colors hover:bg-[var(--elevated)] disabled:opacity-50"
                  onClick={() => void pickCollection(c.id, c.name)}
                >
                  <span className="truncate">{c.name}</span>
                  <span className="shrink-0 text-[11px] text-[var(--subtle)]">
                    {c.itemCount}
                  </span>
                </button>
              ))}
              {collections.length > 0 && (
                <div className="my-1 border-t border-[var(--border)]" />
              )}
              {creating ? (
                <div className="flex flex-col gap-1.5 px-2 py-1.5">
                  <input
                    ref={inputRef}
                    type="text"
                    value={newName}
                    disabled={busy}
                    placeholder="Collection name"
                    className="w-full rounded-md border border-[var(--border)] bg-[var(--elevated)] px-2 py-1.5 text-[12px] text-[var(--mist)] outline-none placeholder:text-[var(--subtle)] focus:border-[var(--lime)]"
                    onChange={(e) => setNewName(e.target.value)}
                    onKeyDown={(e) => {
                      e.stopPropagation()
                      if (e.key === 'Enter') {
                        e.preventDefault()
                        void createAndAdd()
                      }
                    }}
                    onClick={(e) => e.stopPropagation()}
                  />
                  <button
                    type="button"
                    disabled={busy || !newName.trim()}
                    className="rounded-md bg-[var(--lime)] px-2 py-1.5 text-[11px] font-semibold text-[var(--on-lime)] disabled:opacity-50"
                    onClick={() => void createAndAdd()}
                  >
                    Create & add
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  role="menuitem"
                  disabled={busy}
                  className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-[12px] text-[var(--mist)] transition-colors hover:bg-[var(--elevated)] disabled:opacity-50"
                  onClick={() => {
                    setCreating(true)
                    setError(null)
                  }}
                >
                  <Plus size={14} strokeWidth={1.75} />
                  New collection…
                </button>
              )}
              {error && (
                <p className="px-3 py-1.5 text-[11px] text-[rgb(240_120_120)]">{error}</p>
              )}
            </>
          )}
        </div>
      )}
    </>
  )
}
