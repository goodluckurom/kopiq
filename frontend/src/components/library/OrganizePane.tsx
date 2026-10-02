import { useEffect, useState } from 'react'
import {
  Bookmark,
  FolderOpen,
  ListOrdered,
  Pencil,
  Plus,
  Trash2,
  Copy,
  Play,
} from 'lucide-react'
import type { ClipboardItem, ClipboardStack, Collection, NavFilter, Snippet } from '../../lib/types'
import {
  createCollection,
  deleteCollection,
  deleteSnippet,
  deleteStack,
  listCollectionItems,
  listCollections,
  listSnippets,
  listStacks,
  renameCollection,
  saveSnippetFromItem,
  startSavedStack,
} from '../../lib/api'
import { ItemCard } from './ItemCard'
import { ConfirmModal } from '../shared/ConfirmModal'
import { cn } from '../../lib/utils'

type Props = {
  filter: NavFilter
  onFilterChange: (f: NavFilter) => void
  selectedId: number | null
  onSelect: (id: number) => void
  onOpenMenu: (item: ClipboardItem, anchor: { top: number; left: number }) => void
  refreshKey?: number
}

export function OrganizePane({
  filter,
  onFilterChange,
  selectedId,
  onSelect,
  onOpenMenu,
  refreshKey = 0,
}: Props) {
  if (filter.category === 'snippets') {
    return <SnippetsView refreshKey={refreshKey} />
  }
  if (filter.category === 'stacks') {
    return <StacksView refreshKey={refreshKey} />
  }
  if (filter.category === 'collections') {
    return (
      <CollectionsView
        filter={filter}
        onFilterChange={onFilterChange}
        selectedId={selectedId}
        onSelect={onSelect}
        onOpenMenu={onOpenMenu}
        refreshKey={refreshKey}
      />
    )
  }
  return null
}

function SnippetsView({ refreshKey }: { refreshKey: number }) {
  const [snippets, setSnippets] = useState<Snippet[]>([])
  const [busy, setBusy] = useState(false)

  const reload = () => {
    void listSnippets().then(setSnippets).catch(() => setSnippets([]))
  }

  useEffect(() => {
    reload()
  }, [refreshKey])

  return (
    <div className="flex h-full min-w-0 flex-col">
      <div className="shrink-0 px-4 pb-2 pt-4">
        <h2 className="text-[15px] font-semibold text-[var(--mist)]">Snippets</h2>
        <p className="mt-0.5 text-[12px] text-[var(--subtle)]">
          Save from a row menu — optional alias like <code className="text-[var(--muted)]">:go-handler</code>
        </p>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-4">
        {snippets.length === 0 ? (
          <div className="px-2 py-16 text-center">
            <Bookmark className="mx-auto mb-3 text-[var(--subtle)]" size={28} strokeWidth={1.5} />
            <p className="text-[13px] text-[var(--mist)]">No snippets yet.</p>
            <p className="mt-1 text-[12px] text-[var(--subtle)]">
              Right-click a text or code item → Save snippet.
            </p>
          </div>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {snippets.map((sn) => (
              <li
                key={sn.id}
                className="rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5"
              >
                <div className="flex items-start gap-2">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="truncate text-[13px] font-semibold text-[var(--mist)]">
                        {sn.title}
                      </span>
                      {(sn.aliases || []).map((a) => (
                        <span
                          key={a}
                          className="rounded-md bg-[rgb(251_113_133_/0.14)] px-1.5 py-0.5 font-mono text-[10px] font-semibold text-[rgb(244_63_94)]"
                        >
                          :{a}
                        </span>
                      ))}
                      {sn.language && (
                        <span className="text-[10px] uppercase tracking-wide text-[var(--subtle)]">
                          {sn.language}
                        </span>
                      )}
                    </div>
                    <p className="mt-1 line-clamp-2 font-mono text-[11px] text-[var(--muted)]">
                      {sn.body}
                    </p>
                  </div>
                  <div className="flex shrink-0 gap-1">
                    <button
                      type="button"
                      title="Copy body"
                      className="rounded-md p-1.5 text-[var(--subtle)] hover:bg-[var(--elevated-2)] hover:text-[var(--mist)]"
                      onClick={() => {
                        void navigator.clipboard.writeText(sn.body)
                      }}
                    >
                      <Copy size={14} />
                    </button>
                    <button
                      type="button"
                      title="Delete snippet"
                      disabled={busy}
                      className="rounded-md p-1.5 text-[var(--subtle)] hover:bg-[rgb(220_70_70_/0.12)] hover:text-[rgb(240_120_120)]"
                      onClick={() => {
                        setBusy(true)
                        void deleteSnippet(sn.id)
                          .then(reload)
                          .finally(() => setBusy(false))
                      }}
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function StacksView({ refreshKey }: { refreshKey: number }) {
  const [stacks, setStacks] = useState<ClipboardStack[]>([])
  const [busy, setBusy] = useState(false)
  const [toast, setToast] = useState<string | null>(null)

  const reload = () => {
    void listStacks().then(setStacks).catch(() => setStacks([]))
  }

  useEffect(() => {
    reload()
  }, [refreshKey])

  return (
    <div className="flex h-full min-w-0 flex-col">
      <div className="shrink-0 px-4 pb-2 pt-4">
        <h2 className="text-[15px] font-semibold text-[var(--mist)]">Stacks</h2>
        <p className="mt-0.5 text-[12px] text-[var(--subtle)]">
          Multi-select items → Start stack. Re-arm a saved stack here.
        </p>
        {toast && <p className="mt-2 text-[11px] text-[var(--mist)]">{toast}</p>}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-4">
        {stacks.length === 0 ? (
          <div className="px-2 py-16 text-center">
            <ListOrdered
              className="mx-auto mb-3 text-[var(--subtle)]"
              size={28}
              strokeWidth={1.5}
            />
            <p className="text-[13px] text-[var(--mist)]">No saved stacks yet.</p>
            <p className="mt-1 text-[12px] text-[var(--subtle)]">
              ⌘/Ctrl-click items in All Items, then Start stack.
            </p>
          </div>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {stacks.map((st) => (
              <li
                key={st.id}
                className="flex items-center gap-2 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5"
              >
                <div className="min-w-0 flex-1">
                  <div className="truncate text-[13px] font-semibold text-[var(--mist)]">
                    {st.name}
                  </div>
                  <div className="mt-0.5 text-[11px] text-[var(--subtle)]">
                    {st.stepCount} step{st.stepCount === 1 ? '' : 's'}
                  </div>
                </div>
                <button
                  type="button"
                  disabled={busy}
                  className="inline-flex shrink-0 items-center gap-1 rounded-md px-2 py-1.5 text-[11px] font-medium text-[var(--mist)] hover:bg-[var(--elevated-2)]"
                  onClick={() => {
                    setBusy(true)
                    void startSavedStack(st.id)
                      .then((res) => {
                        setToast(res.label ? `Armed · ${res.label}` : 'Stack armed')
                        window.setTimeout(() => setToast(null), 2500)
                      })
                      .catch((e) => {
                        setToast(e instanceof Error ? e.message : 'Failed')
                        window.setTimeout(() => setToast(null), 3500)
                      })
                      .finally(() => setBusy(false))
                  }}
                >
                  <Play size={12} />
                  Start
                </button>
                <button
                  type="button"
                  disabled={busy}
                  title="Delete stack"
                  className="rounded-md p-1.5 text-[var(--subtle)] hover:bg-[rgb(220_70_70_/0.12)] hover:text-[rgb(240_120_120)]"
                  onClick={() => {
                    setBusy(true)
                    void deleteStack(st.id)
                      .then(reload)
                      .finally(() => setBusy(false))
                  }}
                >
                  <Trash2 size={14} />
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function CollectionsView({
  filter,
  onFilterChange,
  selectedId,
  onSelect,
  onOpenMenu,
  refreshKey,
}: Props) {
  const [collections, setCollections] = useState<Collection[]>([])
  const [items, setItems] = useState<ClipboardItem[]>([])
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [renamingId, setRenamingId] = useState<number | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const [renameBusy, setRenameBusy] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<Collection | null>(null)
  const [deleting, setDeleting] = useState(false)

  const reloadCols = () => {
    void listCollections().then(setCollections).catch(() => setCollections([]))
  }

  useEffect(() => {
    reloadCols()
  }, [refreshKey])

  useEffect(() => {
    if (!filter.collectionId) {
      setItems([])
      return
    }
    void listCollectionItems(filter.collectionId)
      .then(setItems)
      .catch(() => setItems([]))
  }, [filter.collectionId, refreshKey])

  const active = collections.find((c) => c.id === filter.collectionId)

  const startRename = (c: Collection) => {
    setRenamingId(c.id)
    setRenameValue(c.name)
  }

  const cancelRename = () => {
    if (renameBusy) return
    setRenamingId(null)
    setRenameValue('')
  }

  const commitRename = async () => {
    if (renamingId == null) return
    const next = renameValue.trim()
    if (!next) {
      cancelRename()
      return
    }
    const current = collections.find((c) => c.id === renamingId)
    if (current && current.name === next) {
      cancelRename()
      return
    }
    setRenameBusy(true)
    try {
      await renameCollection(renamingId, next)
      setRenamingId(null)
      setRenameValue('')
      reloadCols()
      window.dispatchEvent(new CustomEvent('kopiq:counts-changed'))
    } catch (e) {
      console.error('renameCollection failed', e)
    } finally {
      setRenameBusy(false)
    }
  }

  const confirmDelete = async () => {
    if (!pendingDelete) return
    setDeleting(true)
    try {
      await deleteCollection(pendingDelete.id)
      if (filter.collectionId === pendingDelete.id) {
        onFilterChange({ category: 'collections', collectionId: undefined })
      }
      setPendingDelete(null)
      reloadCols()
      window.dispatchEvent(new CustomEvent('kopiq:counts-changed'))
    } catch (e) {
      console.error('deleteCollection failed', e)
    } finally {
      setDeleting(false)
    }
  }

  if (filter.collectionId && active) {
    return (
      <div className="flex h-full min-w-0 flex-col">
        <div className="shrink-0 px-4 pb-2 pt-4">
          <button
            type="button"
            className="text-[11px] font-medium text-[var(--lime)] hover:underline"
            onClick={() =>
              onFilterChange({ category: 'collections', collectionId: undefined })
            }
          >
            ← All collections
          </button>
          <div className="mt-1 flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              {renamingId === active.id ? (
                <input
                  autoFocus
                  value={renameValue}
                  disabled={renameBusy}
                  onChange={(e) => setRenameValue(e.target.value)}
                  className="w-full rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-2.5 py-1.5 text-[15px] font-semibold text-[var(--mist)] outline-none focus:border-[var(--lime)]"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      void commitRename()
                    }
                    if (e.key === 'Escape') cancelRename()
                  }}
                  onBlur={() => void commitRename()}
                />
              ) : (
                <h2 className="text-[15px] font-semibold text-[var(--mist)]">{active.name}</h2>
              )}
              <p className="mt-0.5 text-[12px] text-[var(--subtle)]">
                {items.length} item{items.length === 1 ? '' : 's'}
              </p>
            </div>
            <div className="flex shrink-0 gap-0.5">
              <button
                type="button"
                title="Rename collection"
                className="rounded-md p-1.5 text-[var(--subtle)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]"
                onClick={() =>
                  renamingId === active.id ? void commitRename() : startRename(active)
                }
              >
                <Pencil size={14} />
              </button>
              <button
                type="button"
                title="Delete collection"
                className="rounded-md p-1.5 text-[var(--subtle)] hover:bg-[rgb(220_70_70_/0.12)] hover:text-[rgb(240_120_120)]"
                onClick={() => setPendingDelete(active)}
              >
                <Trash2 size={14} />
              </button>
            </div>
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-4">
          {items.length === 0 ? (
            <p className="px-2 py-12 text-center text-[13px] text-[var(--subtle)]">
              Empty. Add items from a row menu → Add to collection.
            </p>
          ) : (
            <ul className="flex flex-col gap-1">
              {items.map((item) => (
                <li key={item.id}>
                  <ItemCard
                    item={item}
                    selected={selectedId === item.id}
                    onSelect={() => onSelect(item.id)}
                    onOpenMenu={(anchor) => onOpenMenu(item, anchor)}
                  />
                </li>
              ))}
            </ul>
          )}
        </div>
        <ConfirmModal
          open={!!pendingDelete}
          title={`Delete “${pendingDelete?.name ?? ''}”?`}
          description="The collection is removed. Clipboard items stay in history."
          confirmLabel="Delete"
          danger
          busy={deleting}
          onCancel={() => {
            if (!deleting) setPendingDelete(null)
          }}
          onConfirm={() => void confirmDelete()}
        />
      </div>
    )
  }

  return (
    <div className="flex h-full min-w-0 flex-col">
      <div className="flex shrink-0 items-start justify-between gap-2 px-4 pb-2 pt-4">
        <div>
          <h2 className="text-[15px] font-semibold text-[var(--mist)]">Collections</h2>
          <p className="mt-0.5 text-[12px] text-[var(--subtle)]">
            File copies on purpose — not automatic groups.
          </p>
        </div>
        <button
          type="button"
          className="inline-flex items-center gap-1 rounded-[8px] bg-[var(--lime)] px-2.5 py-1.5 text-[12px] font-semibold text-[var(--on-lime)]"
          onClick={() => setCreating(true)}
        >
          <Plus size={14} />
          New
        </button>
      </div>

      {creating && (
        <div className="mx-4 mb-3 flex gap-2">
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Collection name"
            className="min-w-0 flex-1 rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-2.5 py-1.5 text-[12px] text-[var(--mist)] outline-none focus:border-[var(--lime)]"
            onKeyDown={(e) => {
              if (e.key === 'Enter' && name.trim()) {
                void createCollection(name.trim()).then(() => {
                  setName('')
                  setCreating(false)
                  reloadCols()
                  window.dispatchEvent(new CustomEvent('kopiq:counts-changed'))
                })
              }
              if (e.key === 'Escape') setCreating(false)
            }}
          />
          <button
            type="button"
            className="rounded-[8px] bg-[var(--elevated-2)] px-3 py-1.5 text-[12px] font-medium text-[var(--mist)]"
            onClick={() => {
              if (!name.trim()) return
              void createCollection(name.trim()).then(() => {
                setName('')
                setCreating(false)
                reloadCols()
                window.dispatchEvent(new CustomEvent('kopiq:counts-changed'))
              })
            }}
          >
            Create
          </button>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto px-3 pb-4">
        {collections.length === 0 ? (
          <div className="px-2 py-16 text-center">
            <FolderOpen className="mx-auto mb-3 text-[var(--subtle)]" size={28} strokeWidth={1.5} />
            <p className="text-[13px] text-[var(--mist)]">No collections yet.</p>
            <p className="mt-1 text-[12px] text-[var(--subtle)]">
              Create one, then add items from the row menu.
            </p>
          </div>
        ) : (
          <ul className="flex flex-col gap-1">
            {collections.map((c) => (
              <li key={c.id} className="group flex items-center gap-1">
                {renamingId === c.id ? (
                  <input
                    autoFocus
                    value={renameValue}
                    disabled={renameBusy}
                    className="min-w-0 flex-1 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[13px] font-medium text-[var(--mist)] outline-none focus:border-[var(--lime)]"
                    onChange={(e) => setRenameValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault()
                        void commitRename()
                      }
                      if (e.key === 'Escape') cancelRename()
                    }}
                    onBlur={() => void commitRename()}
                  />
                ) : (
                  <button
                    type="button"
                    className={cn(
                      'flex min-w-0 flex-1 items-center gap-3 rounded-[10px] px-3 py-2.5 text-left transition-colors',
                      'hover:bg-[var(--elevated)]',
                    )}
                    onClick={() =>
                      onFilterChange({ category: 'collections', collectionId: c.id })
                    }
                  >
                    <FolderOpen size={18} className="shrink-0 text-[rgb(45_212_191)]" />
                    <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-[var(--mist)]">
                      {c.name}
                    </span>
                    <span className="text-[11px] text-[var(--subtle)]">{c.itemCount}</span>
                  </button>
                )}
                <button
                  type="button"
                  title="Rename collection"
                  className="rounded-md p-1.5 text-[var(--subtle)] opacity-60 hover:bg-[var(--elevated)] hover:text-[var(--mist)] group-hover:opacity-100"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    if (renamingId === c.id) void commitRename()
                    else startRename(c)
                  }}
                >
                  <Pencil size={14} />
                </button>
                <button
                  type="button"
                  title="Delete collection"
                  className="rounded-md p-1.5 text-[var(--subtle)] opacity-60 hover:bg-[rgb(220_70_70_/0.12)] hover:text-[rgb(240_120_120)] group-hover:opacity-100"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    setPendingDelete(c)
                  }}
                >
                  <Trash2 size={14} />
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <ConfirmModal
        open={!!pendingDelete}
        title={`Delete “${pendingDelete?.name ?? ''}”?`}
        description="The collection is removed. Clipboard items stay in history."
        confirmLabel="Delete"
        danger
        busy={deleting}
        onCancel={() => {
          if (!deleting) setPendingDelete(null)
        }}
        onConfirm={() => void confirmDelete()}
      />
    </div>
  )
}

/** Prompt helper used by row menus. */
export async function promptSaveSnippet(item: ClipboardItem): Promise<void> {
  const title =
    window.prompt('Snippet title', item.title || item.textPreview?.slice(0, 40) || 'Snippet') ||
    ''
  if (!title.trim()) return
  const alias =
    window.prompt('Optional alias (e.g. go-handler). Leave blank for none.', '') || ''
  await saveSnippetFromItem(item.id, title.trim(), alias.trim())
}
