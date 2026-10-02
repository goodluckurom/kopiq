import { cn, isFileItem, typeLabel } from '../../lib/utils'
import type { ClipboardItem } from '../../lib/types'
import {
  Braces,
  Code2,
  FileText,
  Image as ImageIcon,
  Link2,
  File,
  Mail,
  Terminal,
  KeyRound,
  Database,
  Palette,
} from 'lucide-react'
import { ItemImage } from './ItemImage'
import { UrlFavicon } from './UrlFavicon'
import { fileVisualFromItem } from '../../lib/fileIcons'
import {
  fileIconDataUrlFromExtension,
  fileIconDataUrlFromItem,
  fileIconDataUrlFromPath,
} from '../../lib/fileIconAssets'

const iconMap: Record<string, typeof Braces> = {
  json: Braces,
  code: Code2,
  typescript: Code2,
  javascript: Code2,
  go: Code2,
  python: Code2,
  image: ImageIcon,
  screenshot: ImageIcon,
  url: Link2,
  email: Mail,
  text: FileText,
  file: File,
  files: File,
  shell: Terminal,
  bash: Terminal,
  jwt: KeyRound,
  sql: Database,
  color: Palette,
}

/** Map detected language / type → file extension for logo lookup. */
const LANG_EXT: Record<string, string> = {
  typescript: 'ts',
  javascript: 'js',
  tsx: 'tsx',
  jsx: 'jsx',
  go: 'go',
  python: 'py',
  rust: 'rs',
  java: 'java',
  kotlin: 'kt',
  swift: 'swift',
  ruby: 'rb',
  php: 'php',
  csharp: 'cs',
  'c#': 'cs',
  cpp: 'cpp',
  'c++': 'cpp',
  c: 'c',
  sql: 'sql',
  shell: 'sh',
  bash: 'sh',
  zsh: 'zsh',
  json: 'json',
  yaml: 'yaml',
  yml: 'yml',
  toml: 'toml',
  html: 'html',
  css: 'css',
  scss: 'scss',
  vue: 'vue',
  svelte: 'svelte',
  dart: 'dart',
  lua: 'lua',
  r: 'r',
  scala: 'scala',
  elixir: 'ex',
  erlang: 'erl',
  haskell: 'hs',
  zig: 'zig',
  nim: 'nim',
  solid: 'sol',
  solidity: 'sol',
  graphql: 'graphql',
  proto: 'proto',
  markdown: 'md',
  md: 'md',
  jwt: 'pem',
  code: 'txt',
}

function isCodeLike(item: ClipboardItem): boolean {
  const t = (item.detectedType || '').toLowerCase()
  const lang = (item.language || '').toLowerCase()
  if (['code', 'json', 'sql', 'shell', 'jwt', 'yaml', 'xml', 'csv'].includes(t)) {
    return true
  }
  if (lang && LANG_EXT[lang]) return true
  return false
}

function codeLogoSrc(item: ClipboardItem): string | null {
  const path = item.context?.filePath || item.context?.fileName || ''
  if (path && /\.[a-z0-9]+$/i.test(path.split(/[/\\]/).pop() || '')) {
    return fileIconDataUrlFromPath(path)
  }
  const lang = (item.language || '').toLowerCase()
  if (lang && LANG_EXT[lang]) {
    return fileIconDataUrlFromExtension(LANG_EXT[lang])
  }
  const t = (item.detectedType || '').toLowerCase()
  if (t && LANG_EXT[t]) {
    return fileIconDataUrlFromExtension(LANG_EXT[t])
  }
  const line = (item.textContent || item.textPreview || '').split('\n')[0] || ''
  if (line.startsWith('#!') && line.includes('python')) {
    return fileIconDataUrlFromExtension('py')
  }
  if (line.startsWith('#!') && (line.includes('bash') || line.includes('sh'))) {
    return fileIconDataUrlFromExtension('sh')
  }
  if (t === 'json' || line.trimStart().startsWith('{') || line.trimStart().startsWith('[')) {
    return fileIconDataUrlFromExtension('json')
  }
  return fileIconDataUrlFromExtension('txt')
}

type Props = {
  item: ClipboardItem
  size?: 'sm' | 'md' | 'lg'
  className?: string
  showLabel?: boolean
}

export function TypeBadge({ item, size = 'sm', className, showLabel }: Props) {
  const detected = (item.detectedType || '').toLowerCase()
  const key = (detected || item.language || item.kind).toLowerCase()
  const label = typeLabel(item)

  const box =
    size === 'lg'
      ? 'h-10 w-10 rounded-[10px]'
      : size === 'md'
        ? 'h-9 w-9 rounded-[9px]'
        : 'h-8 w-8 rounded-lg'

  const iconSize = size === 'lg' ? 22 : size === 'md' ? 20 : 16
  const px = size === 'lg' ? 40 : size === 'md' ? 36 : 32

  if (item.detectedType === 'color' && item.textContent) {
    return (
      <div
        className={cn(
          'flex shrink-0 items-center justify-center border border-[var(--border)]',
          box,
          className,
        )}
        style={{ background: item.textContent }}
        title={label}
      />
    )
  }

  if (
    item.kind === 'image' ||
    item.detectedType === 'image' ||
    item.detectedType === 'screenshot'
  ) {
    return (
      <ItemImage
        itemId={item.id}
        kind="thumb"
        width={px}
        height={px}
        revision={item.thumbnailPath || ''}
        className={cn(
          'shrink-0 overflow-hidden border border-[var(--border)]',
          box,
          className,
        )}
      />
    )
  }

  // Prefer detected type for email / similar labels so language never shadows the icon.
  if (detected === 'email') {
    return (
      <div className={cn('flex items-center gap-1.5', className)}>
        <div
          className={cn(
            'flex shrink-0 items-center justify-center border border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)]',
            box,
          )}
          title={label}
        >
          <Mail size={iconSize} strokeWidth={1.75} />
        </div>
        {showLabel && (
          <span className="rounded-md bg-[var(--elevated-2)] px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-[var(--muted)]">
            {label}
          </span>
        )}
      </div>
    )
  }

  if (isFileItem(item)) {
    const visual = fileVisualFromItem(item)
    const src = fileIconDataUrlFromItem(item)
    return (
      <div className={cn('flex items-center gap-1.5', className)}>
        <div
          className={cn(
            'flex shrink-0 items-center justify-center border border-[var(--border)] bg-[var(--elevated)]',
            box,
          )}
          title={`${visual.label} · ${visual.category}`}
        >
          <img
            src={src}
            alt={visual.label}
            width={iconSize}
            height={iconSize}
            className="object-contain"
            draggable={false}
          />
        </div>
        {showLabel && (
          <span className="rounded-md bg-[var(--elevated-2)] px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-[var(--muted)]">
            {visual.label}
          </span>
        )}
      </div>
    )
  }

  if (isCodeLike(item)) {
    const src = codeLogoSrc(item)
    const pill = (item.language || item.detectedType || 'CODE').toUpperCase()
    return (
      <div className={cn('flex items-center gap-1.5', className)}>
        <div
          className={cn(
            'flex shrink-0 items-center justify-center border border-[var(--border)] bg-[var(--elevated)]',
            box,
          )}
          title={pill}
        >
          {src ? (
            <img
              src={src}
              alt={pill}
              width={iconSize}
              height={iconSize}
              className="object-contain"
              draggable={false}
            />
          ) : (
            <Code2 size={iconSize} strokeWidth={1.75} />
          )}
        </div>
        {showLabel && (
          <span className="rounded-md bg-[var(--elevated-2)] px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-[var(--muted)]">
            {label}
          </span>
        )}
      </div>
    )
  }

  if (detected === 'url') {
    const urlText = item.textContent || item.textPreview || ''
    return (
      <div className={cn('flex items-center gap-1.5', className)}>
        <div
          className={cn(
            'flex shrink-0 items-center justify-center border border-[var(--border)] bg-[var(--elevated)] text-[var(--mist)]',
            box,
          )}
          title={label}
        >
          <UrlFavicon urlText={urlText} size={iconSize} title={label} />
        </div>
        {showLabel && (
          <span className="rounded-md bg-[var(--elevated-2)] px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-[var(--muted)]">
            {label}
          </span>
        )}
      </div>
    )
  }

  const Icon = iconMap[key] || FileText

  return (
    <div className={cn('flex items-center gap-1.5', className)}>
      <div
        className={cn(
          'flex shrink-0 items-center justify-center bg-[var(--elevated-2)] text-[var(--mist)] border border-[var(--border)]',
          box,
        )}
        title={label}
      >
        <Icon size={iconSize} strokeWidth={1.75} />
      </div>
      {showLabel && (
        <span className="rounded-md bg-[var(--elevated-2)] px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-[var(--muted)]">
          {label}
        </span>
      )}
    </div>
  )
}

export function TypePill({ item, className }: { item: ClipboardItem; className?: string }) {
  const file = isFileItem(item)
  const visual = file ? fileVisualFromItem(item) : null
  const code = !file && isCodeLike(item)
  const codeLabel = code
    ? (item.language || item.detectedType || 'code').toUpperCase()
    : null
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-md px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide',
        file || code
          ? 'border border-[var(--border)]'
          : 'bg-[var(--elevated-2)] text-[var(--muted)]',
        className,
      )}
      style={
        visual
          ? { background: visual.bg, color: visual.color }
          : code
            ? { background: 'rgb(79 70 229 / 0.12)', color: '#4f46e5' }
            : undefined
      }
    >
      {visual ? visual.label : codeLabel || typeLabel(item)}
    </span>
  )
}
