import { useEffect, useState } from 'react'
import { Link2 } from 'lucide-react'
import { cn } from '../../lib/utils'

const failedHosts = new Set<string>()
const okCache = new Map<string, string>()

/** Extract hostname from a copied URL string. */
export function hostnameFromUrlText(text: string): string | null {
  const raw = text.trim().split(/\s/)[0] || ''
  if (!raw) return null
  try {
    const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `https://${raw}`
    const u = new URL(withScheme)
    if (u.protocol !== 'http:' && u.protocol !== 'https:') return null
    const host = u.hostname.replace(/^www\./i, '')
    if (!host || !host.includes('.')) return null
    return host.toLowerCase()
  } catch {
    return null
  }
}

function faviconCandidates(host: string, size: number): string[] {
  const sz = size <= 16 ? 16 : size <= 32 ? 32 : 64
  return [
    `https://www.google.com/s2/favicons?domain=${encodeURIComponent(host)}&sz=${sz}`,
    `https://icons.duckduckgo.com/ip3/${encodeURIComponent(host)}.ico`,
  ]
}

type Props = {
  urlText: string
  size: number
  className?: string
  title?: string
}

/**
 * Shows a site favicon when online / loadable; falls back to the Link icon.
 */
export function UrlFavicon({ urlText, size, className, title }: Props) {
  const host = hostnameFromUrlText(urlText)
  const [src, setSrc] = useState<string | null>(() =>
    host ? okCache.get(host) ?? null : null,
  )
  const [failed, setFailed] = useState(() => (host ? failedHosts.has(host) : true))

  useEffect(() => {
    if (!host) {
      setSrc(null)
      setFailed(true)
      return
    }
    if (failedHosts.has(host)) {
      setSrc(null)
      setFailed(true)
      return
    }
    const cached = okCache.get(host)
    if (cached) {
      setSrc(cached)
      setFailed(false)
      return
    }

    let cancelled = false
    const candidates = faviconCandidates(host, size)
    let i = 0

    const tryNext = () => {
      if (cancelled || i >= candidates.length) {
        if (!cancelled) {
          failedHosts.add(host)
          setSrc(null)
          setFailed(true)
        }
        return
      }
      const url = candidates[i++]
      const img = new Image()
      img.onload = () => {
        if (cancelled) return
        // Google returns a tiny generic globe (16x16) for unknowns; accept it —
        // still better than a plain link glyph for known sites. On total miss we
        // already fall through candidates.
        okCache.set(host, url)
        setSrc(url)
        setFailed(false)
      }
      img.onerror = () => {
        if (!cancelled) tryNext()
      }
      img.src = url
    }

    setFailed(false)
    setSrc(null)
    tryNext()

    return () => {
      cancelled = true
    }
  }, [host, size])

  if (!host || failed || !src) {
    return (
      <Link2
        size={size}
        strokeWidth={1.75}
        className={className}
        aria-label={title || 'Link'}
      />
    )
  }

  return (
    <img
      src={src}
      alt={title || host}
      width={size}
      height={size}
      className={cn('object-contain', className)}
      draggable={false}
      onError={() => {
        failedHosts.add(host)
        setSrc(null)
        setFailed(true)
      }}
    />
  )
}
