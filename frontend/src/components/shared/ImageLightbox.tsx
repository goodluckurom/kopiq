import { useEffect, useRef } from 'react'
import { X, Maximize2 } from 'lucide-react'
import { ItemImage } from './ItemImage'
import { cn } from '../../lib/utils'

type Props = {
  open: boolean
  itemId: number
  title?: string
  kind?: 'thumb' | 'original'
  revision?: string
  meta?: string
  onClose: () => void
}

export function ImageLightbox({
  open,
  itemId,
  title,
  kind = 'original',
  revision,
  meta,
  onClose,
}: Props) {
  const closeRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    const prev = document.activeElement as HTMLElement | null
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
      prev?.focus?.()
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[90] flex flex-col bg-[rgb(0_0_0_/0.78)] animate-fade-in"
      role="dialog"
      aria-modal="true"
      aria-label={title || 'Image preview'}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex shrink-0 items-center gap-3 px-4 py-3">
        <div className="min-w-0 flex-1">
          {title && (
            <p className="truncate text-[13px] font-medium text-white/90">{title}</p>
          )}
          {meta && (
            <p className="truncate text-[11px] text-white/50">{meta}</p>
          )}
        </div>
        <button
          ref={closeRef}
          type="button"
          aria-label="Close"
          onClick={onClose}
          className="rounded-lg p-2 text-white/70 hover:bg-white/10 hover:text-white"
        >
          <X size={18} />
        </button>
      </div>
      <div
        className="flex min-h-0 flex-1 items-center justify-center p-4 pt-0"
        onMouseDown={(e) => {
          if (e.target === e.currentTarget) onClose()
        }}
      >
        <ItemImage
          itemId={itemId}
          kind={kind}
          revision={revision}
          fit="contain"
          className="max-h-full max-w-full rounded-lg shadow-[0_24px_80px_rgb(0_0_0_/0.5)]"
        />
      </div>
    </div>
  )
}

type EnlargeBtnProps = {
  onClick: () => void
  className?: string
}

export function EnlargeImageButton({ onClick, className }: EnlargeBtnProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-2.5 py-1.5 text-[11px] font-medium text-[var(--mist)] transition-colors hover:bg-[var(--elevated-2)]',
        className,
      )}
    >
      <Maximize2 size={12} />
      Enlarge
    </button>
  )
}
