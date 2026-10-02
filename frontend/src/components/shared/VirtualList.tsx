import { useEffect, useRef, type ReactNode, type RefObject } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { cn } from '../../lib/utils'

type Props = {
  count: number
  estimateSize: (index: number) => number
  overscan?: number
  className?: string
  children: (index: number) => ReactNode
  scrollToIndex?: number | null
  scrollAlign?: 'start' | 'center' | 'end' | 'auto'
  /** Optional external scroll parent (e.g. Quick Paste). */
  scrollParentRef?: RefObject<HTMLDivElement | null>
  'aria-label'?: string
}

/** Windowed list for library / Quick Paste (checklist §21). */
export function VirtualList({
  count,
  estimateSize,
  overscan = 10,
  className,
  children,
  scrollToIndex,
  scrollAlign = 'auto',
  scrollParentRef,
  'aria-label': ariaLabel,
}: Props) {
  const localRef = useRef<HTMLDivElement>(null)
  const parentRef = scrollParentRef ?? localRef
  const virtualizer = useVirtualizer({
    count,
    getScrollElement: () => parentRef.current,
    estimateSize,
    overscan,
  })

  useEffect(() => {
    if (scrollToIndex == null || scrollToIndex < 0 || scrollToIndex >= count) return
    virtualizer.scrollToIndex(scrollToIndex, { align: scrollAlign })
  }, [scrollToIndex, scrollAlign, count, virtualizer])

  const rows = virtualizer.getVirtualItems()

  return (
    <div
      ref={parentRef as RefObject<HTMLDivElement>}
      className={cn('min-h-0 flex-1 overflow-y-auto', className)}
      role={ariaLabel ? 'listbox' : undefined}
      aria-label={ariaLabel}
    >
      <div
        className="relative w-full"
        style={{ height: `${virtualizer.getTotalSize()}px` }}
      >
        {rows.map((row) => (
          <div
            key={row.key}
            data-index={row.index}
            ref={virtualizer.measureElement}
            className="absolute left-0 top-0 w-full"
            style={{ transform: `translateY(${row.start}px)` }}
          >
            {children(row.index)}
          </div>
        ))}
      </div>
    </div>
  )
}
