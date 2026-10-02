import { useEffect, useState } from 'react'
import { Image as ImageIcon } from 'lucide-react'
import { cn } from '../../lib/utils'

const urlCache = new Map<string, string>()

async function loadItemImage(
  id: number,
  kind: 'thumb' | 'original',
): Promise<string | null> {
  const key = `${id}:${kind}`
  const hit = urlCache.get(key)
  if (hit) return hit
  try {
    const mod = await import('../../../bindings/kopiq/service/historyservice')
    if (!mod.ItemImageDataURL) return null
    const url = await mod.ItemImageDataURL(id, kind)
    if (url) {
      urlCache.set(key, url)
      return url
    }
  } catch {
    // bindings / runtime unavailable
  }
  return null
}

/** Drop cached URLs when history updates so new thumbs appear. */
export function invalidateItemImage(id: number) {
  urlCache.delete(`${id}:thumb`)
  urlCache.delete(`${id}:original`)
}

type Props = {
  itemId: number
  kind?: 'thumb' | 'original'
  width?: number
  height?: number
  className?: string
  /** object-fit; defaults to cover for thumbs, contain for lightbox use. */
  fit?: 'cover' | 'contain'
  /** When thumbnailPath arrives after async gen, bump to refetch. */
  revision?: string
}

export function ItemImage({
  itemId,
  kind = 'thumb',
  width,
  height,
  className,
  fit = 'cover',
  revision,
}: Props) {
  const [src, setSrc] = useState<string | null>(() =>
    urlCache.get(`${itemId}:${kind}`) ?? null,
  )

  useEffect(() => {
    let cancelled = false
    if (revision) {
      urlCache.delete(`${itemId}:${kind}`)
    }
    const cached = urlCache.get(`${itemId}:${kind}`)
    if (cached && !revision) {
      setSrc(cached)
      return
    }
    setSrc(null)
    void loadItemImage(itemId, kind).then((url) => {
      if (!cancelled) setSrc(url)
    })
    return () => {
      cancelled = true
    }
  }, [itemId, kind, revision])

  if (src) {
    return (
      <img
        src={src}
        alt=""
        width={width}
        height={height}
        className={cn(fit === 'contain' ? 'object-contain' : 'object-cover', className)}
        draggable={false}
      />
    )
  }

  return (
    <div
      className={cn(
        'flex items-center justify-center bg-[var(--elevated-2)] text-[var(--muted)]',
        className,
      )}
      style={{ width, height }}
    >
      <ImageIcon size={Math.min(18, (width || 36) * 0.45)} />
    </div>
  )
}
