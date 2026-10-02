import { useEffect, useState } from 'react'
import { formatRelativeTime } from '../../lib/utils'
import { cn } from '../../lib/utils'

type Props = {
  timestamp: number
  className?: string
}

export function RelativeTime({ timestamp, className }: Props) {
  const [now, setNow] = useState(Date.now())

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 15_000)
    return () => window.clearInterval(id)
  }, [])

  return (
    <time
      className={cn('tabular-nums text-[var(--subtle)]', className)}
      dateTime={new Date(timestamp).toISOString()}
      title={new Date(timestamp).toLocaleString()}
    >
      {formatRelativeTime(timestamp, now)}
    </time>
  )
}
