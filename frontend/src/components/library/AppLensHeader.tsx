import type { ClipboardItem, NavFilter } from '../../lib/types'
import { AppIcon } from '../shared/AppIcon'
import { cn } from '../../lib/utils'

type Props = {
  filter: NavFilter
  /** All history items (unfiltered) for count tallies. */
  allItems: ClipboardItem[]
  onFilterChange: (f: NavFilter) => void
}

function countFor(
  items: ClipboardItem[],
  app: string,
  kind: NonNullable<NavFilter['lensType']> | 'all' | 'today' | 'week',
): number {
  const appLower = app.toLowerCase()
  const fromApp = items.filter((i) =>
    (i.context?.applicationName || '').toLowerCase().includes(appLower),
  )
  const now = Date.now()
  const d = new Date(now)
  const startToday = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const wd = d.getDay() === 0 ? 7 : d.getDay()
  const startWeek = startToday - (wd - 1) * 86400000

  return fromApp.filter((item) => {
    if (kind === 'all') return true
    if (kind === 'today') return item.lastCopiedAt >= startToday
    if (kind === 'week') return item.lastCopiedAt >= startWeek
    switch (kind) {
      case 'text':
        return item.detectedType === 'text' || item.detectedType === 'color'
      case 'images':
        return (
          item.detectedType === 'image' ||
          item.detectedType === 'screenshot' ||
          item.kind === 'image'
        )
      case 'links':
        return item.detectedType === 'url'
      case 'code':
        return ['code', 'json', 'sql', 'shell', 'jwt', 'yaml', 'xml', 'csv'].includes(
          item.detectedType,
        )
      case 'files':
        return (
          item.detectedType === 'file' ||
          item.detectedType === 'files' ||
          item.kind === 'files'
        )
      default:
        return true
    }
  }).length
}

export function AppLensHeader({ filter, allItems, onFilterChange }: Props) {
  const app = filter.sourceApp
  if (!app) return null

  const total = countFor(allItems, app, 'all')
  const iconItem = allItems.find((i) =>
    (i.context?.applicationName || '').toLowerCase().includes(app.toLowerCase()),
  )
  const iconKey =
    iconItem?.context?.iconCacheKey || iconItem?.context?.applicationId

  const chips: {
    id: string
    label: string
    active: boolean
    onClick: () => void
    count: number
  }[] = [
    {
      id: 'text',
      label: 'Text',
      active: filter.lensType === 'text',
      count: countFor(allItems, app, 'text'),
      onClick: () =>
        onFilterChange({
          ...filter,
          lensType: filter.lensType === 'text' ? undefined : 'text',
        }),
    },
    {
      id: 'images',
      label: 'Images',
      active: filter.lensType === 'images',
      count: countFor(allItems, app, 'images'),
      onClick: () =>
        onFilterChange({
          ...filter,
          lensType: filter.lensType === 'images' ? undefined : 'images',
        }),
    },
    {
      id: 'links',
      label: 'Links',
      active: filter.lensType === 'links',
      count: countFor(allItems, app, 'links'),
      onClick: () =>
        onFilterChange({
          ...filter,
          lensType: filter.lensType === 'links' ? undefined : 'links',
        }),
    },
    {
      id: 'code',
      label: 'Code',
      active: filter.lensType === 'code',
      count: countFor(allItems, app, 'code'),
      onClick: () =>
        onFilterChange({
          ...filter,
          lensType: filter.lensType === 'code' ? undefined : 'code',
        }),
    },
    {
      id: 'files',
      label: 'Files',
      active: filter.lensType === 'files',
      count: countFor(allItems, app, 'files'),
      onClick: () =>
        onFilterChange({
          ...filter,
          lensType: filter.lensType === 'files' ? undefined : 'files',
        }),
    },
    {
      id: 'today',
      label: 'Today',
      active: filter.lensPeriod === 'today',
      count: countFor(allItems, app, 'today'),
      onClick: () =>
        onFilterChange({
          ...filter,
          lensPeriod: filter.lensPeriod === 'today' ? undefined : 'today',
        }),
    },
    {
      id: 'week',
      label: 'This week',
      active: filter.lensPeriod === 'week',
      count: countFor(allItems, app, 'week'),
      onClick: () =>
        onFilterChange({
          ...filter,
          lensPeriod: filter.lensPeriod === 'week' ? undefined : 'week',
        }),
    },
  ]

  return (
    <div className="shrink-0 border-b border-[var(--border)] px-4 pb-3 pt-4">
      <button
        type="button"
        className="text-[11px] font-medium text-[var(--lime)] hover:underline"
        onClick={() =>
          onFilterChange({
            category: 'all',
            sourceApp: undefined,
            lensType: undefined,
            lensPeriod: undefined,
          })
        }
      >
        ← All items
      </button>
      <div className="mt-2 flex items-center gap-3">
        <AppIcon iconKey={iconKey} name={app} size={28} />
        <div className="min-w-0">
          <h2 className="truncate text-[15px] font-semibold text-[var(--mist)]">{app}</h2>
          <p className="text-[12px] text-[var(--subtle)]">
            {total} clipboard item{total === 1 ? '' : 's'}
          </p>
        </div>
      </div>
      <div className="mt-3 flex flex-wrap gap-1.5">
        {chips.map((c) => (
          <button
            key={c.id}
            type="button"
            onClick={c.onClick}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-[11px] font-medium transition-colors',
              c.active
                ? 'border-[var(--lime)] bg-[var(--lime-soft)] text-[var(--lime)]'
                : 'border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)] hover:bg-[var(--elevated-2)]',
            )}
          >
            {c.label}
            <span className="text-[10px] opacity-70">{c.count}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
