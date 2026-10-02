import {
  useEffect,
  useMemo,
  useState,
  type MouseEvent as ReactMouseEvent,
  type Ref,
  type RefObject,
} from 'react'
import { ChevronDown, ChevronRight, Layers, Trash2 } from 'lucide-react'
import type { ClipboardItem, NavFilter } from '../../lib/types'
import {
  multiCopy,
  stackStatus,
  startStack,
  stopStack,
} from '../../lib/api'
import {
  clusterByContext,
  contextClusterFooter,
  emptyStateFor,
  groupByTime,
  placeLabel,
  type ContextCluster,
} from '../../lib/utils'
import { AppIcon } from '../shared/AppIcon'
import { VirtualList } from '../shared/VirtualList'
import { AppLensHeader } from './AppLensHeader'
import { ItemCard } from './ItemCard'
import { SearchBar } from './SearchBar'

const COPY_SHAPES = [
  { id: 'plain' as const, label: 'Plain text' },
  { id: 'lines' as const, label: 'Lines' },
  { id: 'comma' as const, label: 'Comma separated' },
  { id: 'csv' as const, label: 'CSV' },
  { id: 'json' as const, label: 'JSON array' },
  { id: 'markdown' as const, label: 'Markdown' },
]

type Props = {
  items: ClipboardItem[]
  allItems?: ClipboardItem[]
  selectedId: number | null
  checkedIds: number[]
  query: string
  filter: NavFilter
  onQueryChange: (q: string) => void
  onFilterChange?: (f: NavFilter) => void
  onSelect: (id: number) => void
  onCheckedChange: (ids: number[]) => void
  onOpenMenu: (item: ClipboardItem, anchor: { top: number; left: number }) => void
  onRequestDelete?: () => void
  onRequestClearAll?: () => void
  searchRef?: RefObject<HTMLInputElement | null>
  onToast?: (msg: string) => void
}

type FlatRow =
  | { kind: 'time'; key: string; label: string }
  | { kind: 'cluster'; key: string; cluster: ContextCluster; collapsed: boolean }
  | { kind: 'item'; key: string; item: ClipboardItem }
  | { kind: 'cluster-pad'; key: string }

function isImageLike(item: ClipboardItem): boolean {
  return (
    item.kind === 'image' ||
    item.detectedType === 'image' ||
    item.detectedType === 'screenshot'
  )
}

function estimateRow(row: FlatRow): number {
  switch (row.kind) {
    case 'time':
      return 28
    case 'cluster':
      return 44
    case 'cluster-pad':
      return 6
    case 'item':
      return isImageLike(row.item) ? 92 : 68
  }
}

export function ItemList({
  items,
  allItems,
  selectedId,
  checkedIds,
  query,
  filter,
  onQueryChange,
  onFilterChange,
  onSelect,
  onCheckedChange,
  onOpenMenu,
  onRequestDelete,
  onRequestClearAll,
  searchRef,
  onToast,
}: Props) {
  const canClear = items.some((i) => !i.isPinned)
  const empty = emptyStateFor(filter, query)
  const groups = useMemo(() => groupByTime(items), [items])
  const useContextGroups =
    filter.category === 'all' && !filter.sourceApp && !query.trim()
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [copyOpen, setCopyOpen] = useState(false)
  const [stackLabel, setStackLabel] = useState<string | null>(null)
  const checked = useMemo(() => new Set(checkedIds), [checkedIds])

  useEffect(() => {
    let cancelled = false
    const tick = () => {
      void stackStatus().then((st) => {
        if (cancelled) return
        setStackLabel(st.armed ? st.label || 'Stack armed' : null)
      })
    }
    tick()
    const id = window.setInterval(tick, 1500)
    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [])

  const rows = useMemo(() => {
    const out: FlatRow[] = []
    for (const group of groups) {
      out.push({ kind: 'time', key: `t:${group.id}`, label: group.label })
      const defaultCollapsed = group.id === 'older'
      if (!useContextGroups) {
        for (const item of group.items) {
          out.push({ kind: 'item', key: `i:${item.id}`, item })
        }
        continue
      }
      for (const cluster of clusterByContext(group.items)) {
        if (cluster.items.length === 1) {
          out.push({
            kind: 'item',
            key: `i:${cluster.items[0].id}`,
            item: cluster.items[0],
          })
          continue
        }
        const isCollapsed = collapsed[cluster.key] ?? defaultCollapsed
        out.push({
          kind: 'cluster',
          key: `c:${cluster.key}`,
          cluster,
          collapsed: isCollapsed,
        })
        if (!isCollapsed) {
          for (const item of cluster.items) {
            out.push({ kind: 'item', key: `i:${item.id}`, item })
          }
          out.push({ kind: 'cluster-pad', key: `p:${cluster.key}` })
        }
      }
    }
    return out
  }, [groups, useContextGroups, collapsed])

  const estimateSize = useMemo(
    () => (index: number) => estimateRow(rows[index]!),
    [rows],
  )

  const toggleCheck = (id: number, e?: ReactMouseEvent) => {
    if (e?.metaKey || e?.ctrlKey) {
      const next = new Set(checked)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      onCheckedChange([...next])
      onSelect(id)
      return
    }
    if (e?.shiftKey && selectedId != null) {
      const ids = items.map((i) => i.id)
      const a = ids.indexOf(selectedId)
      const b = ids.indexOf(id)
      if (a >= 0 && b >= 0) {
        const [lo, hi] = a < b ? [a, b] : [b, a]
        onCheckedChange(ids.slice(lo, hi + 1))
        onSelect(id)
        return
      }
    }
    onCheckedChange([])
    onSelect(id)
  }

  return (
    <div className="flex h-full min-w-0 flex-col">
      {filter.sourceApp && onFilterChange && (
        <AppLensHeader
          filter={filter}
          allItems={allItems || items}
          onFilterChange={onFilterChange}
        />
      )}
      <div className="shrink-0 px-4 pb-3 pt-4">
        <SearchBar
          value={query}
          onChange={onQueryChange}
          inputRef={searchRef as Ref<HTMLInputElement> | undefined}
        />
      </div>

      <div className="mb-2 flex flex-wrap items-center justify-end gap-1 px-4">
        {stackLabel && (
          <>
            <span className="mr-auto text-[11px] text-[var(--mist)]">{stackLabel}</span>
            <button
              type="button"
              className="rounded-md px-2 py-1 text-[11px] font-medium text-[var(--subtle)] hover:bg-[var(--elevated)]"
              onClick={() => {
                void stopStack().then(() => {
                  setStackLabel(null)
                  onToast?.('Stack stopped')
                })
              }}
            >
              Stop stack
            </button>
          </>
        )}
        {checkedIds.length > 0 && (
          <>
            <span className="mr-auto text-[11px] text-[var(--subtle)]">
              {checkedIds.length} selected · ⌘/Ctrl-click to toggle
            </span>
            <div className="relative">
              <button
                type="button"
                className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-[var(--mist)] hover:bg-[var(--elevated)]"
                onClick={() => setCopyOpen((o) => !o)}
              >
                Copy as…
              </button>
              {copyOpen && (
                <div className="absolute right-0 z-20 mt-1 min-w-[160px] rounded-[10px] border border-[var(--border)] bg-[var(--surface)] py-1 shadow-lg">
                  {COPY_SHAPES.map((s) => (
                    <button
                      key={s.id}
                      type="button"
                      className="block w-full px-3 py-1.5 text-left text-[12px] text-[var(--mist)] hover:bg-[var(--elevated)]"
                      onClick={() => {
                        setCopyOpen(false)
                        void multiCopy(checkedIds, s.id)
                          .then((r) => {
                            onToast?.(
                              r.note
                                ? `Copied ${r.itemsUsed} · ${r.note}`
                                : `Copied ${r.itemsUsed} as ${s.label}`,
                            )
                          })
                          .catch((e) =>
                            onToast?.(e instanceof Error ? e.message : 'Copy failed'),
                          )
                      }}
                    >
                      {s.label}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <button
              type="button"
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-[var(--mist)] hover:bg-[var(--elevated)]"
              onClick={() => {
                void startStack(checkedIds)
                  .then((st) => {
                    onCheckedChange([])
                    onToast?.(st.label ? `Armed · ${st.label}` : 'Stack armed')
                  })
                  .catch((e) =>
                    onToast?.(e instanceof Error ? e.message : 'Stack failed'),
                  )
              }}
            >
              <Layers size={12} />
              Start stack
            </button>
            <button
              type="button"
              className="rounded-md px-2 py-1 text-[11px] font-medium text-[var(--subtle)] hover:bg-[var(--elevated)]"
              onClick={() => onCheckedChange([])}
            >
              Clear selection
            </button>
          </>
        )}
        {selectedId != null && onRequestDelete && checkedIds.length === 0 && (
          <button
            type="button"
            className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium text-[var(--subtle)] hover:bg-[var(--elevated)] hover:text-[rgb(240_120_120)]"
            onClick={onRequestDelete}
          >
            <Trash2 size={12} />
            Remove
          </button>
        )}
        {canClear && onRequestClearAll && (
          <button
            type="button"
            className="rounded-md px-2 py-1 text-[11px] font-medium text-[var(--subtle)] hover:bg-[var(--elevated)] hover:text-[rgb(240_120_120)]"
            onClick={onRequestClearAll}
          >
            Clear all
          </button>
        )}
      </div>

      {items.length === 0 ? (
        <div className="flex flex-col items-center px-4 py-16 text-center">
          <p className="text-[13px] text-[var(--mist)]">{empty.message}</p>
          {empty.action && (
            <p className="mt-1.5 text-[12px] text-[var(--subtle)]">{empty.action}</p>
          )}
        </div>
      ) : (
        <VirtualList
          count={rows.length}
          estimateSize={estimateSize}
          overscan={12}
          className="px-3 pb-4"
          aria-label="Clipboard history"
        >
          {(index) => {
            const row = rows[index]!
            if (row.kind === 'time') {
              return (
                <h2 className="mb-1.5 px-1 pt-2 text-[12px] font-semibold uppercase tracking-[0.06em] text-[var(--subtle)] first:pt-0">
                  {row.label}
                </h2>
              )
            }
            if (row.kind === 'cluster') {
              const { cluster, collapsed: isCollapsed } = row
              const place = cluster.place || placeLabel(cluster.items[0]) || ''
              return (
                <div className="mb-1 overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--elevated)]/50">
                  <button
                    type="button"
                    className="flex w-full items-center gap-2 px-2.5 py-2 text-left hover:bg-[var(--elevated)]"
                    onClick={() =>
                      setCollapsed((prev) => ({
                        ...prev,
                        [cluster.key]: !isCollapsed,
                      }))
                    }
                  >
                    {isCollapsed ? (
                      <ChevronRight size={14} className="text-[var(--subtle)]" />
                    ) : (
                      <ChevronDown size={14} className="text-[var(--subtle)]" />
                    )}
                    <AppIcon
                      name={cluster.appName}
                      iconKey={cluster.iconKey}
                      size={18}
                    />
                    <span className="min-w-0 flex-1 truncate text-[12px] font-medium text-[var(--mist)]">
                      {cluster.appName}
                      {place ? (
                        <span className="font-normal text-[var(--subtle)]">
                          {' '}
                          · {place}
                        </span>
                      ) : null}
                    </span>
                    <span className="shrink-0 text-[10px] text-[var(--subtle)]">
                      {contextClusterFooter(cluster)}
                    </span>
                  </button>
                </div>
              )
            }
            if (row.kind === 'cluster-pad') {
              return <div className="h-1" />
            }
            return (
              <div className="mb-1" role="option">
                <ItemCard
                  item={row.item}
                  selected={row.item.id === selectedId}
                  checked={checked.has(row.item.id)}
                  onSelect={(e) => toggleCheck(row.item.id, e)}
                  onOpenMenu={(anchor) => onOpenMenu(row.item, anchor)}
                />
              </div>
            )
          }}
        </VirtualList>
      )}
    </div>
  )
}
