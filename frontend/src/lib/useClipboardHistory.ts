import { useCallback, useEffect, useState } from 'react'
import { listItems, searchItems } from './api'
import type { ClipboardItem } from './types'

function canUseEvents(): boolean {
  if (typeof window === 'undefined') return false
  const w = window as Window & { wails?: unknown; go?: unknown }
  return Boolean(w.wails || w.go)
}

/**
 * Loads clipboard history and keeps it fresh when the Go pipeline emits
 * `history.changed` (new copy, bump, pin, delete).
 */
export function useClipboardHistory(query = '') {
  const [items, setItems] = useState<ClipboardItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(async () => {
    try {
      const data = query.trim()
        ? await searchItems(query)
        : await listItems()
      setItems(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load history')
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [query])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    void (async () => {
      await reload()
      if (cancelled) return
    })()
    return () => {
      cancelled = true
    }
  }, [reload])

  useEffect(() => {
    if (!canUseEvents()) return
    let off: (() => void) | undefined
    let alive = true

    void import('@wailsio/runtime')
      .then(({ Events }) => {
        if (!alive) return
        off = Events.On('history.changed', () => {
          void reload()
        })
      })
      .catch(() => {
        // runtime not available (browser preview)
      })

    return () => {
      alive = false
      off?.()
    }
  }, [reload])

  return { items, setItems, loading, error, reload }
}
