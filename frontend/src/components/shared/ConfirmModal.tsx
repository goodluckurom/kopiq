import { useEffect, useRef, type ReactNode } from 'react'
import { cn } from '../../lib/utils'

type Props = {
  open: boolean
  title: string
  description?: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  busy?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmModal({
  open,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger,
  busy,
  onConfirm,
  onCancel,
}: Props) {
  const confirmRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    const prev = document.activeElement as HTMLElement | null
    confirmRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onCancel()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('keydown', onKey)
      prev?.focus?.()
    }
  }, [open, onCancel])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[80] flex items-center justify-center bg-[rgb(0_0_0_/0.45)] p-4 animate-fade-in"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !busy) onCancel()
      }}
    >
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="kopiq-confirm-title"
        aria-describedby={description ? 'kopiq-confirm-desc' : undefined}
        className="w-full max-w-[360px] rounded-[14px] border border-[var(--border)] bg-[var(--surface)] p-5 shadow-[0_16px_48px_rgb(0_0_0_/0.35)]"
      >
        <h2
          id="kopiq-confirm-title"
          className="text-[15px] font-semibold text-[var(--mist)]"
        >
          {title}
        </h2>
        {description && (
          <p
            id="kopiq-confirm-desc"
            className="mt-2 text-[13px] leading-relaxed text-[var(--muted)]"
          >
            {description}
          </p>
        )}
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={onCancel}
            className="rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3.5 py-2 text-[12px] font-medium text-[var(--mist)] transition-colors hover:bg-[var(--elevated-2)] disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            ref={confirmRef}
            type="button"
            disabled={busy}
            onClick={onConfirm}
            className={cn(
              'rounded-[10px] px-3.5 py-2 text-[12px] font-semibold transition-opacity hover:opacity-90 disabled:opacity-50',
              danger
                ? 'bg-[rgb(220_70_70)] text-white'
                : 'bg-[var(--lime)] text-[var(--on-lime)]',
            )}
          >
            {busy ? 'Working…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
