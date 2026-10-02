import type { ClipboardItem } from '../../lib/types'
import { cn, formatBytes, placeLabel } from '../../lib/utils'
import { TypeBadge, TypePill } from '../shared/TypeBadge'
import { AppIcon } from '../shared/AppIcon'
import { RelativeTime } from '../shared/RelativeTime'
import { ItemImage } from '../shared/ItemImage'
import { MoreHorizontal } from 'lucide-react'
import type { MouseEvent as ReactMouseEvent } from 'react'

type Props = {
  item: ClipboardItem
  selected?: boolean
  checked?: boolean
  onSelect: (e?: ReactMouseEvent) => void
  onDoubleClick?: () => void
  onOpenMenu?: (anchor: { top: number; left: number }) => void
  compact?: boolean
}

function imageMetaLine(item: ClipboardItem): string {
  const parts: string[] = []
  if (item.width && item.height) {
    parts.push(`${item.width} × ${item.height}`)
  }
  const fmt = (item.mimeType || '').split('/')[1]?.toUpperCase()
  if (fmt) parts.push(fmt === 'JPEG' ? 'JPEG' : fmt)
  if (item.byteSize) parts.push(formatBytes(item.byteSize))
  if (item.imageKept === false) parts.push('preview only')
  return parts.join(' · ') || item.textPreview
}

export function ItemCard({
  item,
  selected,
  checked,
  onSelect,
  onDoubleClick,
  onOpenMenu,
  compact,
}: Props) {
  const place = placeLabel(item)
  const app = item.context?.applicationName
  const isImage =
    item.kind === 'image' ||
    item.detectedType === 'image' ||
    item.detectedType === 'screenshot'
  const line2 =
    item.context?.documentTitle ||
    item.context?.windowTitle ||
    item.context?.fileName ||
    ''

  return (
    <div
      role="option"
      aria-selected={selected}
      tabIndex={0}
      onClick={(e) => onSelect(e)}
      onDoubleClick={onDoubleClick}
      onContextMenu={(e) => {
        e.preventDefault()
        onSelect()
        onOpenMenu?.({ top: e.clientY, left: e.clientX })
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onSelect()
        }
        if (e.key === 'ContextMenu' || (e.shiftKey && e.key === 'F10')) {
          e.preventDefault()
          const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
          onOpenMenu?.({ top: rect.bottom, left: rect.right - 8 })
        }
      }}
      className={cn(
        'group flex w-full cursor-pointer items-start gap-3 rounded-[10px] border px-3 py-2.5 text-left transition-all duration-[120ms] ease-out',
        selected
          ? 'border-[var(--lime)] bg-[var(--lime-soft)]'
          : checked
            ? 'border-[var(--border)] bg-[var(--elevated)]'
            : 'border-transparent bg-transparent hover:bg-[var(--elevated)]',
        compact && 'py-2',
      )}
    >
      {isImage && !compact ? (
        <ItemImage
          itemId={item.id}
          kind="thumb"
          width={72}
          height={54}
          revision={item.thumbnailPath || item.blobPath || ''}
          className="shrink-0 overflow-hidden rounded-[8px] border border-[var(--border)] bg-[var(--elevated-2)]"
        />
      ) : (
        <TypeBadge item={item} size={compact ? 'sm' : 'md'} />
      )}

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-[13px] font-semibold text-[var(--mist)]">
            {item.title}
          </span>
          <TypePill item={item} />
          {item.isPinned && (
            <span className="text-[10px] text-[var(--lime)]">Pinned</span>
          )}
        </div>
        <p className="mt-0.5 truncate font-mono text-[11px] text-[var(--muted)]">
          {isImage
            ? imageMetaLine(item)
            : item.sensitivity === 'secret' || item.sensitivity === 'suspected'
              ? '••••••••••••••••'
              : item.textPreview}
        </p>
        {(app || line2 || place) && !compact && (
          <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-0.5">
            {app && (
              <div className="flex max-w-[180px] items-center gap-1.5">
                <AppIcon
                  name={app}
                  iconKey={item.context?.iconCacheKey || item.context?.applicationId}
                  size={14}
                />
                <span className="truncate text-[11px] text-[var(--subtle)]">{app}</span>
              </div>
            )}
            {line2 && (
              <span className="max-w-[200px] truncate text-[11px] text-[var(--subtle)]">
                {line2}
              </span>
            )}
            {place && place !== line2 && (
              <span className="max-w-[200px] truncate font-mono text-[10px] text-[var(--subtle)] opacity-80">
                {place}
              </span>
            )}
          </div>
        )}
      </div>

      <div className="flex shrink-0 flex-col items-end gap-1 pl-2">
        <RelativeTime
          timestamp={item.lastCopiedAt}
          className="text-[11px]"
        />
        {onOpenMenu && (
          <button
            type="button"
            aria-label="Item actions"
            className="rounded-md p-1 text-[var(--subtle)] opacity-0 transition-opacity hover:bg-[var(--elevated-2)] hover:text-[var(--mist)] group-hover:opacity-100 focus:opacity-100"
            onClick={(e) => {
              e.stopPropagation()
              onSelect()
              const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
              onOpenMenu({ top: rect.bottom + 4, left: rect.right - 200 })
            }}
          >
            <MoreHorizontal size={14} />
          </button>
        )}
      </div>
    </div>
  )
}
