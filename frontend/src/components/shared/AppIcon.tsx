import { useEffect, useState } from 'react'
import { cn } from '../../lib/utils'

const APP_COLORS: Record<string, string> = {
  cursor: '#00C853',
  'google chrome': '#4285F4',
  chrome: '#4285F4',
  'microsoft edge': '#0078D4',
  edge: '#0078D4',
  'vs code': '#007ACC',
  vscode: '#007ACC',
  figma: '#F24E1E',
  terminal: '#8B918F',
  finder: '#4A90E2',
  chatgpt: '#10A37F',
}

const urlCache = new Map<string, string>()

function initials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase()
  return name.slice(0, 2).toUpperCase()
}

async function loadIconURL(key: string): Promise<string | null> {
  if (!key) return null
  const hit = urlCache.get(key)
  if (hit) return hit
  try {
    const mod = await import('../../../bindings/kopiq/service/historyservice')
    if (!mod.AppIconDataURL) return null
    const url = await mod.AppIconDataURL(key)
    if (url) {
      urlCache.set(key, url)
      return url
    }
  } catch {
    // bindings / runtime unavailable
  }
  return null
}

type Props = {
  name: string
  iconKey?: string
  size?: number
  className?: string
}

export function AppIcon({ name, iconKey, size = 16, className }: Props) {
  const cacheKey = iconKey || ''
  const [src, setSrc] = useState<string | null>(
    cacheKey ? urlCache.get(cacheKey) ?? null : null,
  )

  useEffect(() => {
    if (!cacheKey) {
      setSrc(null)
      return
    }
    let cancelled = false
    const cached = urlCache.get(cacheKey)
    if (cached) {
      setSrc(cached)
      return
    }
    void loadIconURL(cacheKey).then((url) => {
      if (!cancelled) setSrc(url)
    })
    return () => {
      cancelled = true
    }
  }, [cacheKey])

  if (src) {
    return (
      <img
        src={src}
        alt=""
        width={size}
        height={size}
        className={cn('inline-block shrink-0 rounded-[4px] object-contain', className)}
        title={name}
        aria-hidden
      />
    )
  }

  const key = name.toLowerCase()
  const color =
    APP_COLORS[key] ||
    Object.entries(APP_COLORS).find(([k]) => key.includes(k))?.[1] ||
    '#6B716F'

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center justify-center rounded-[4px] text-[9px] font-bold text-white',
        className,
      )}
      style={{
        width: size,
        height: size,
        background: color,
        fontSize: Math.max(8, size * 0.45),
      }}
      title={name}
      aria-hidden
    >
      {initials(name)}
    </span>
  )
}
