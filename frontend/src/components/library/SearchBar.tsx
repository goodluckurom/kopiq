import type { Ref } from 'react'
import { Search } from 'lucide-react'
import { searchShortcutLabel } from '../../lib/platform'
import { cn } from '../../lib/utils'

type Props = {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  autoFocus?: boolean
  className?: string
  showShortcut?: boolean
  inputRef?: Ref<HTMLInputElement>
}

export function SearchBar({
  value,
  onChange,
  placeholder = 'Search clipboard... (type, app, domain, content...)',
  autoFocus,
  className,
  showShortcut = true,
  inputRef,
}: Props) {
  return (
    <div
      className={cn(
        'relative flex h-10 items-center gap-2 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 transition-colors duration-[120ms] focus-within:border-[color-mix(in_srgb,var(--lime)_45%,transparent)]',
        className,
      )}
    >
      <Search size={15} className="shrink-0 text-[var(--subtle)]" strokeWidth={1.75} />
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoFocus={autoFocus}
        spellCheck={false}
        className="min-w-0 flex-1 bg-transparent text-[13px] text-[var(--mist)] outline-none placeholder:text-[var(--subtle)]"
      />
      {showShortcut && (
        <kbd className="hidden shrink-0 rounded-md border border-[var(--border)] bg-[var(--surface)] px-1.5 py-0.5 text-[10px] font-medium text-[var(--subtle)] sm:inline">
          {searchShortcutLabel()}
        </kbd>
      )}
    </div>
  )
}
