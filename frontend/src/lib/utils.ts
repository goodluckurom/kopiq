import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'
import type { ClipboardItem, CategoryId, NavFilter } from './types'
import { fileVisualFromItem } from './fileIcons'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function formatRelativeTime(ts: number, now = Date.now()): string {
  const diff = Math.max(0, now - ts)
  const sec = Math.floor(diff / 1000)
  if (sec < 60) return `${sec}s ago`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h ago`
  const days = Math.floor(hr / 24)
  if (days === 1) return 'Yesterday'
  if (days < 7) return `${days}d ago`
  return new Date(ts).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
  })
}

export function typeLabel(item: ClipboardItem): string {
  if (item.id < 0 || item.metadataJson?.includes('"snippet"')) {
    return 'SNIPPET'
  }
  if (isFileItem(item)) {
    return fileVisualFromItem(item).label
  }
  const t = (item.language || item.detectedType || item.kind).toUpperCase()
  if (t === 'TYPESCRIPT') return 'TS'
  if (t === 'JAVASCRIPT') return 'JS'
  if (t === 'SCREENSHOT') return 'Screenshot'
  if (t === 'IMAGE') return 'Image'
  if (t === 'FILES') return 'FILE'
  if (t === 'SNIPPET') return 'SNIPPET'
  return t
}

export function isFileItem(item: ClipboardItem): boolean {
  return (
    item.kind === 'files' ||
    item.detectedType === 'file' ||
    item.detectedType === 'files'
  )
}

export function placeLabel(item: ClipboardItem): string {
  const ctx = item.context
  if (!ctx) return ''
  if (ctx.domain) return ctx.domain + (ctx.url?.includes('/') ? ctx.url.replace(/^https?:\/\/[^/]+/, '').split('?')[0] : '')
  if (ctx.filePath) {
    const parts = ctx.filePath.split('/')
    return parts.slice(-2).join('/')
  }
  if (ctx.fileName) return ctx.fileName
  if (ctx.windowTitle) return ctx.windowTitle
  return ''
}

export function matchesFilter(item: ClipboardItem, filter: NavFilter): boolean {
  if (filter.tag && !(item.tags || []).includes(filter.tag)) return false
  if (filter.sourceApp) {
    const name = item.context?.applicationName?.toLowerCase() ?? ''
    if (!name.includes(filter.sourceApp.toLowerCase())) return false
  }
  if (filter.lensPeriod) {
    const now = Date.now()
    const d = new Date(now)
    const startToday = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
    if (filter.lensPeriod === 'today' && item.lastCopiedAt < startToday) return false
    if (filter.lensPeriod === 'week') {
      const wd = d.getDay() === 0 ? 7 : d.getDay()
      const startWeek = startToday - (wd - 1) * 86400000
      if (item.lastCopiedAt < startWeek) return false
    }
  }
  if (filter.lensType) {
    switch (filter.lensType) {
      case 'text':
        if (!(item.detectedType === 'text' || item.detectedType === 'color')) return false
        break
      case 'images':
        if (
          !(
            item.detectedType === 'image' ||
            item.detectedType === 'screenshot' ||
            item.kind === 'image'
          )
        )
          return false
        break
      case 'links':
        if (item.detectedType !== 'url') return false
        break
      case 'code':
        if (
          !['code', 'json', 'sql', 'shell', 'jwt', 'yaml', 'xml', 'csv'].includes(
            item.detectedType,
          )
        )
          return false
        break
      case 'files':
        if (
          !(
            item.detectedType === 'file' ||
            item.detectedType === 'files' ||
            item.kind === 'files'
          )
        )
          return false
        break
    }
  }
  switch (filter.category) {
    case 'all':
      return true
    case 'pinned':
      return item.isPinned
    case 'images':
      return (
        item.detectedType === 'image' ||
        item.detectedType === 'screenshot' ||
        item.kind === 'image'
      )
    case 'code':
      return ['code', 'json', 'sql', 'shell', 'jwt', 'yaml', 'xml', 'csv'].includes(
        item.detectedType,
      )
    case 'links':
      return item.detectedType === 'url'
    case 'text':
      return (
        item.detectedType === 'text' ||
        item.detectedType === 'color' ||
        item.detectedType === 'email' ||
        item.detectedType === 'uuid' ||
        item.detectedType === 'ip' ||
        item.detectedType === 'timestamp'
      )
    case 'files':
      return (
        item.detectedType === 'file' ||
        item.detectedType === 'files' ||
        item.kind === 'files'
      )
    case 'snippets':
    case 'collections':
    case 'stacks':
      return false
    default:
      return true
  }
}

export function matchesQuery(item: ClipboardItem, query: string): boolean {
  const q = query.trim().toLowerCase()
  if (!q) return true
  const hay = [
    item.title,
    item.textPreview,
    item.textContent,
    item.detectedType,
    item.language,
    item.context?.applicationName,
    item.context?.domain,
    item.context?.url,
    item.context?.filePath,
    item.context?.fileName,
    item.context?.documentTitle,
    ...(item.tags || []),
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
  return hay.includes(q)
}

export function categoryTitle(id: CategoryId): string {
  const map: Record<CategoryId, string> = {
    all: 'All Items',
    pinned: 'Pinned',
    images: 'Images',
    code: 'Code',
    links: 'Links',
    text: 'Text',
    files: 'Files',
    snippets: 'Snippets',
    collections: 'Collections',
    stacks: 'Stacks',
  }
  return map[id]
}

export type EmptyStateCopy = {
  message: string
  action?: string
}

/** One-sentence empty states per sidebar filter (KQ-013). */
export function emptyStateFor(
  filter: NavFilter,
  query: string,
): EmptyStateCopy {
  if (query.trim()) {
    return { message: 'Nothing copied matches that.' }
  }
  if (filter.sourceApp) {
    return {
      message: `Nothing from ${filter.sourceApp} yet.`,
      action: 'Copy something while that app is frontmost.',
    }
  }
  switch (filter.category) {
    case 'pinned':
      return {
        message: 'Pin items you want to keep forever.',
        action: 'Use the row menu or detail pane to pin.',
      }
    case 'images':
      return {
        message: 'Screenshots you copy will show up here.',
        action: 'Capture a screenshot or copy an image.',
      }
    case 'code':
      return {
        message: 'Code and JSON you copy will land here.',
        action: 'Copy a snippet from your editor.',
      }
    case 'links':
      return {
        message: 'URLs you copy will show up here.',
        action: 'Copy a link from the browser.',
      }
    case 'text':
      return {
        message: 'Plain text copies will show up here.',
        action: 'Copy something from any app.',
      }
    case 'files':
      return {
        message: 'File paths you copy will show up here.',
        action: 'Copy a path, or copy files from Finder or Explorer.',
      }
    case 'snippets':
      return {
        message: 'Save text or code as a snippet from the row menu.',
        action: 'Right-click an item → Save snippet.',
      }
    case 'collections':
      return {
        message: 'Create a collection to file copies on purpose.',
        action: 'Open Collections and click New.',
      }
    case 'stacks':
      return {
        message: 'No saved stacks yet.',
        action: 'Multi-select items, then Start stack.',
      }
    default:
      return {
        message: 'Copy something — it will show up here with its source app.',
      }
  }
}

export type TimeGroupId = 'today' | 'yesterday' | 'week' | 'older'

export type TimeGroup = {
  id: TimeGroupId
  label: string
  items: ClipboardItem[]
}

/** Bucket items by lastCopiedAt into Today / Yesterday / This Week / Older. */
export function groupByTime(
  items: ClipboardItem[],
  now = Date.now(),
): TimeGroup[] {
  const d = new Date(now)
  const startToday = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const startYesterday = startToday - 86400000
  const wd = d.getDay() === 0 ? 7 : d.getDay()
  const startWeek = startToday - (wd - 1) * 86400000

  const buckets: Record<TimeGroupId, ClipboardItem[]> = {
    today: [],
    yesterday: [],
    week: [],
    older: [],
  }
  for (const item of items) {
    const t = item.lastCopiedAt
    if (t >= startToday) buckets.today.push(item)
    else if (t >= startYesterday) buckets.yesterday.push(item)
    else if (t >= startWeek) buckets.week.push(item)
    else buckets.older.push(item)
  }

  const labels: { id: TimeGroupId; label: string }[] = [
    { id: 'today', label: 'Today' },
    { id: 'yesterday', label: 'Yesterday' },
    { id: 'week', label: 'This Week' },
    { id: 'older', label: 'Older' },
  ]
  return labels
    .map(({ id, label }) => ({ id, label, items: buckets[id] }))
    .filter((g) => g.items.length > 0)
}

export type ContextCluster = {
  key: string
  appName: string
  appId: string
  iconKey?: string
  place: string
  items: ClipboardItem[]
  from: number
  to: number
}

const GROUP_GAP_MS = 3 * 60 * 1000

function sameLocalDay(a: number, b: number): boolean {
  const da = new Date(a)
  const db = new Date(b)
  return (
    da.getFullYear() === db.getFullYear() &&
    da.getMonth() === db.getMonth() &&
    da.getDate() === db.getDate()
  )
}

/**
 * Cluster items that share app + place within a 3-minute gap (KQ-007).
 * Input should already be within one calendar-day bucket.
 * Singles stay as one-item clusters (render without chrome).
 */
export function clusterByContext(items: ClipboardItem[]): ContextCluster[] {
  if (items.length === 0) return []
  // Walk oldest → newest for clustering, then reverse for display.
  const chronological = [...items].sort((a, b) => a.lastCopiedAt - b.lastCopiedAt)
  const clusters: ContextCluster[] = []

  for (const item of chronological) {
    const appId = item.context?.applicationId || item.context?.applicationName || ''
    const place =
      item.context?.domain ||
      item.context?.filePath ||
      item.context?.windowTitle ||
      ''
    const prev = clusters[clusters.length - 1]
    const sameApp = prev && prev.appId === appId
    const samePlace = prev && prev.place === place
    const gapOk = prev && item.lastCopiedAt - prev.to <= GROUP_GAP_MS
    const dayOk = prev && sameLocalDay(item.lastCopiedAt, prev.to)
    if (prev && sameApp && samePlace && gapOk && dayOk) {
      if (!prev.items.some((i) => i.id === item.id)) {
        prev.items.push(item)
      }
      prev.to = Math.max(prev.to, item.lastCopiedAt)
      prev.from = Math.min(prev.from, item.lastCopiedAt)
      continue
    }
    clusters.push({
      key: `${appId}|${place}|${item.id}`,
      appName: item.context?.applicationName || 'Unknown',
      appId,
      iconKey: item.context?.iconCacheKey || item.context?.applicationId,
      place,
      items: [item],
      from: item.lastCopiedAt,
      to: item.lastCopiedAt,
    })
  }

  // Newest clusters first; items inside newest-first.
  return clusters
    .reverse()
    .map((c) => ({
      ...c,
      items: [...c.items].sort((a, b) => b.lastCopiedAt - a.lastCopiedAt),
    }))
}

function formatSpan(from: number, to: number): string {
  const fmt = (t: number) =>
    new Date(t).toLocaleTimeString(undefined, {
      hour: 'numeric',
      minute: '2-digit',
    })
  if (from === to) return fmt(from)
  return `${fmt(from)}–${fmt(to)}`
}

export function contextClusterFooter(c: ContextCluster): string {
  const n = c.items.length
  return `${n} item${n === 1 ? '' : 's'} · ${formatSpan(c.from, c.to)}`
}

/** Simple JSON / code highlighter for preview panes */
export function highlightCode(code: string, language?: string): string {
  const escape = (s: string) =>
    s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

  if (language === 'json' || (!language && code.trim().startsWith('{'))) {
    return escape(code)
      .replace(
        /(&quot;[^&]*&quot;)(\s*:)/g,
        '<span class="tok-key">$1</span>$2',
      )
      .replace(
        /:\s*(&quot;[^&]*&quot;)/g,
        ': <span class="tok-str">$1</span>',
      )
      .replace(
        /:\s*(-?\d+\.?\d*)/g,
        ': <span class="tok-num">$1</span>',
      )
      .replace(
        /:\s*(true|false|null)/g,
        ': <span class="tok-bool">$1</span>',
      )
  }

  let out = escape(code)
  const goKw =
    language === 'go'
      ? /\b(package|import|func|return|if|else|for|range|switch|case|default|type|struct|interface|map|chan|go|defer|select|var|const|make|len|cap|nil|true|false|error)\b/g
      : null
  if (goKw) {
    out = out.replace(goKw, '<span class="tok-kw">$1</span>')
  } else {
    out = out.replace(
      /\b(export|async|function|const|let|var|return|if|else|try|catch|from|import|def|class|extends|implements|SELECT|FROM|JOIN|WHERE|ORDER|BY|LIMIT|ON|AS|INSERT|UPDATE|DELETE|CREATE)\b/gi,
      '<span class="tok-kw">$1</span>',
    )
  }
  out = out.replace(
    /(&quot;[^&]*&quot;|'[^']*'|`[^`]*`)/g,
    '<span class="tok-str">$1</span>',
  )
  out = out.replace(/\b(\d+)\b/g, '<span class="tok-num">$1</span>')
  return out
}
