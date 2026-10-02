import { useEffect, useState, type ReactNode } from 'react'
import {
  Pin,
  MoreHorizontal,
  X,
  Copy,
  ChevronDown,
  ChevronUp,
  BookmarkPlus,
  Braces,
  FolderOpen,
  ExternalLink,
  Globe,
  Clock,
  FileText,
  AppWindow,
  Hash,
  Trash2,
} from 'lucide-react'
import type { ClipboardItem } from '../../lib/types'
import {
  cn,
  formatBytes,
  formatRelativeTime,
  highlightCode,
  isFileItem,
  typeLabel,
} from '../../lib/utils'
import { TypeBadge } from '../shared/TypeBadge'
import { AppIcon } from '../shared/AppIcon'
import { ItemImage } from '../shared/ItemImage'
import { ConfirmModal } from '../shared/ConfirmModal'
import { ImageLightbox, EnlargeImageButton } from '../shared/ImageLightbox'
import {
  applyTransform,
  applyTransformAndCopy,
  copyItem,
  deleteItem,
  keepItem,
  listTransformActions,
  openContextFile,
  openFile,
  renameItem,
  revealFile,
  revealImage,
  saveImage,
  type TransformAction,
} from '../../lib/api'

type Tab = 'preview' | 'context' | 'actions' | 'history'

type Props = {
  item: ClipboardItem | null
  onClose: () => void
  onTogglePin?: () => void
  onDeleted?: (id: number) => void
}

export function DetailPane({ item, onClose, onTogglePin, onDeleted }: Props) {
  const [tab, setTab] = useState<Tab>('preview')
  const [copied, setCopied] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [lightboxOpen, setLightboxOpen] = useState(false)
  const [transformText, setTransformText] = useState<string | null>(null)
  const [transformNote, setTransformNote] = useState<string | null>(null)
  const [transformError, setTransformError] = useState<string | null>(null)
  const [transformBusy, setTransformBusy] = useState(false)
  const [actions, setActions] = useState<TransformAction[]>([])

  // Reset enlarge / delete UI when selection changes.
  useEffect(() => {
    setLightboxOpen(false)
    setConfirmDelete(false)
    setDeleting(false)
    setTransformText(null)
    setTransformNote(null)
    setTransformError(null)
    setActions([])
  }, [item?.id])

  useEffect(() => {
    if (!item?.id) return
    let cancelled = false
    void listTransformActions(item.id).then((a) => {
      if (!cancelled) setActions(a)
    })
    return () => {
      cancelled = true
    }
  }, [item?.id])

  if (!item) {
    return (
      <aside className="flex h-full w-[380px] shrink-0 flex-col items-center justify-center border-l border-[var(--border)] bg-[var(--surface)] px-6 text-center">
        <p className="text-[13px] text-[var(--subtle)]">
          Select an item to preview
        </p>
      </aside>
    )
  }

  const handleCopy = async () => {
    await copyItem(item.id)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1500)
  }

  const handleDelete = async () => {
    setDeleting(true)
    try {
      await deleteItem(item.id)
      setConfirmDelete(false)
      onDeleted?.(item.id)
    } catch {
      setDeleting(false)
    }
  }

  const rawContent = item.textContent || item.textPreview
  const HIGHLIGHT_CAP = 200_000
  const baseContent =
    rawContent.length > HIGHLIGHT_CAP
      ? rawContent.slice(0, HIGHLIGHT_CAP)
      : rawContent
  const content = transformText ?? baseContent
  const lines = content.split('\n')
  const lang =
    transformNote?.includes('Go')
      ? 'go'
      : transformNote?.includes('TypeScript')
        ? 'typescript'
        : item.language || (item.detectedType === 'json' ? 'json' : undefined)
  const isFile = isFileItem(item)
  const contextPath = showContextFilePath(
    item.context?.filePath,
    item.context?.url,
  )
    ? item.context!.filePath!.trim()
    : ''
  const canJSON = actions.some((a) => a.id.startsWith('json.'))

  const runTransform = async (id: string, copy = false) => {
    setTransformBusy(true)
    setTransformError(null)
    try {
      const res = copy
        ? await applyTransformAndCopy(item.id, id)
        : await applyTransform(item.id, id)
      if (res.kind === 'error' || res.error) {
        setTransformError(res.error || 'Transform failed')
        return
      }
      if (res.text) {
        setTransformText(res.text)
        setTransformNote(res.label + (copy ? ' · copied' : ''))
      }
    } catch (e) {
      setTransformError(e instanceof Error ? e.message : String(e))
    } finally {
      setTransformBusy(false)
    }
  }

  const handleOpenFile = async () => {
    try {
      await openFile(item.id)
    } catch (e) {
      // ignore — OS may report missing path
      console.warn(e)
    }
  }

  const handleRevealFile = async () => {
    try {
      await revealFile(item.id)
    } catch (e) {
      console.warn(e)
    }
  }

  const handleOpenInEditor = async () => {
    try {
      await openContextFile(item.id)
    } catch (e) {
      console.warn(e)
    }
  }

  return (
    <aside className="flex h-full w-[400px] shrink-0 flex-col border-l border-[var(--border)] bg-[var(--surface)] animate-fade-in">
      <ConfirmModal
        open={confirmDelete}
        title="Remove this item?"
        description="It will be deleted from your clipboard history. This cannot be undone."
        confirmLabel="Remove"
        danger
        busy={deleting}
        onCancel={() => {
          if (!deleting) setConfirmDelete(false)
        }}
        onConfirm={() => void handleDelete()}
      />
      <ImageLightbox
        open={lightboxOpen}
        itemId={item.id}
        title={item.title}
        kind={item.imageKept === false ? 'thumb' : 'original'}
        revision={(item.thumbnailPath || '') + (item.blobPath || '')}
        meta={
          [
            item.width && item.height ? `${item.width} × ${item.height}` : null,
            (item.mimeType || '').split('/')[1]?.toUpperCase(),
            item.byteSize ? formatBytes(item.byteSize) : null,
          ]
            .filter(Boolean)
            .join(' · ') || undefined
        }
        onClose={() => setLightboxOpen(false)}
      />

      {/* Header */}
      <div className="flex items-start gap-3 border-b border-[var(--border)] px-4 py-3">
        <TypeBadge item={item} size="lg" />
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-[14px] font-semibold text-[var(--mist)]">
            {item.title}
          </h2>
          <p className="mt-0.5 text-[11px] text-[var(--subtle)]">
            {typeLabel(item)} · {formatBytes(item.byteSize)} · Copied{' '}
            {formatRelativeTime(item.lastCopiedAt)}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-0.5">
          {(item.sensitivity !== 'suspected' && item.sensitivity !== 'secret') ||
          item.isPinned ? (
            <IconBtn
              label={item.isPinned ? 'Unpin' : 'Pin'}
              onClick={onTogglePin}
              active={item.isPinned}
            >
              <Pin size={15} strokeWidth={item.isPinned ? 2.25 : 1.75} />
            </IconBtn>
          ) : null}
          <IconBtn label="Remove" onClick={() => setConfirmDelete(true)}>
            <Trash2 size={15} />
          </IconBtn>
          <IconBtn label="More">
            <MoreHorizontal size={15} />
          </IconBtn>
          <IconBtn label="Close" onClick={onClose}>
            <X size={15} />
          </IconBtn>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-4 border-b border-[var(--border)] px-4">
        {(['preview', 'context', 'actions', 'history'] as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={cn(
              'relative py-2.5 text-[12px] font-medium capitalize transition-colors duration-[120ms]',
              tab === t
                ? 'text-[var(--mist)]'
                : 'text-[var(--subtle)] hover:text-[var(--muted)]',
            )}
          >
            {t}
            {tab === t && (
              <span className="absolute inset-x-0 -bottom-px h-0.5 rounded-full bg-[var(--lime)]" />
            )}
          </button>
        ))}
      </div>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {tab === 'preview' && (
          <PreviewTab
            item={item}
            lines={lines}
            lang={lang}
            content={content}
            onEnlarge={() => setLightboxOpen(true)}
            fill
            canJSON={canJSON}
            actions={actions}
            transformNote={transformNote}
            transformError={transformError}
            transformBusy={transformBusy}
            onFormat={() => void runTransform('json.pretty')}
            onMinify={() => void runTransform('json.minify')}
            onTransform={(id) => void runTransform(id)}
            onTransformCopy={(id) => void runTransform(id, true)}
            onResetTransform={() => {
              setTransformText(null)
              setTransformNote(null)
              setTransformError(null)
            }}
          />
        )}
        {tab === 'context' && (
          <div className="min-h-0 flex-1 overflow-y-auto">
            <ContextTab item={item} />
          </div>
        )}
        {tab === 'actions' && (
          <div className="min-h-0 flex-1 overflow-y-auto">
            <ActionsTab item={item} onRequestDelete={() => setConfirmDelete(true)} />
          </div>
        )}
        {tab === 'history' && (
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-6 text-[13px] text-[var(--subtle)]">
            Copied {item.copyCount} time{item.copyCount === 1 ? '' : 's'}. Full
            history lands with later milestones.
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="shrink-0 space-y-3 border-t border-[var(--border)] px-4 py-3">
        <div className="flex gap-2">
          <button
            type="button"
            onClick={handleCopy}
            className="flex flex-1 items-center justify-center gap-2 rounded-[10px] bg-[var(--lime)] px-3 py-2.5 text-[13px] font-semibold text-[var(--on-lime)] transition-opacity duration-[120ms] hover:opacity-90"
          >
            <Copy size={14} strokeWidth={2.25} />
            {copied ? 'Copied' : 'Copy'}
          </button>
          {isFile && (
            <button
              type="button"
              onClick={() => void handleOpenFile()}
              className="flex items-center gap-1.5 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[12px] font-medium text-[var(--mist)] transition-colors hover:bg-[var(--elevated-2)]"
            >
              <ExternalLink size={13} />
              Open
            </button>
          )}
          {isFile && (
            <button
              type="button"
              onClick={() => void handleRevealFile()}
              className="flex items-center gap-1.5 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[12px] font-medium text-[var(--mist)] transition-colors hover:bg-[var(--elevated-2)]"
            >
              <FolderOpen size={13} />
              Reveal
            </button>
          )}
          {!isFile && contextPath && (
            <button
              type="button"
              onClick={() => void handleOpenInEditor()}
              className="flex items-center gap-1.5 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[12px] font-medium text-[var(--mist)] transition-colors hover:bg-[var(--elevated-2)]"
            >
              <ExternalLink size={13} />
              Open in editor
            </button>
          )}
          {!isFile && !contextPath && (
            <button
              type="button"
              className="flex items-center gap-1.5 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[12px] font-medium text-[var(--mist)] transition-colors hover:bg-[var(--elevated-2)]"
            >
              Copy as
              <ChevronDown size={13} />
            </button>
          )}
          <button
            type="button"
            className="flex items-center gap-1.5 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2.5 text-[12px] font-medium text-[var(--mist)] transition-colors hover:bg-[var(--elevated-2)]"
          >
            <BookmarkPlus size={14} />
            Save to Snippet
          </button>
        </div>

        {item.context && (
          <div className="overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] p-3">
            <div className="mb-2 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <AppIcon
                  name={item.context.applicationName}
                  iconKey={item.context.iconCacheKey || item.context.applicationId}
                  size={20}
                />
                <span className="text-[12px] font-medium text-[var(--mist)]">
                  {item.context.applicationName}
                </span>
              </div>
              {item.context.url && (
                <button
                  type="button"
                  className="flex items-center gap-1 text-[11px] font-medium text-[var(--lime)] hover:underline"
                >
                  Open in {item.context.applicationName.split(' ').pop()}
                  <ExternalLink size={11} />
                </button>
              )}
            </div>
            <dl className="space-y-1.5 text-[11px]">
              {item.context.documentTitle && (
                <CtxRow icon={<FileText size={11} />} label="Page Title">
                  <CollapsibleText
                    value={item.context.documentTitle}
                    className="leading-snug"
                  />
                </CtxRow>
              )}
              {item.context.domain && (
                <CtxRow icon={<Hash size={11} />} label="Domain">
                  <span className="block break-all">{item.context.domain}</span>
                </CtxRow>
              )}
              {showContextFilePath(item.context.filePath, item.context.url) && (
                <CtxRow icon={<FileText size={11} />} label="File">
                  <CollapsibleText
                    value={item.context.filePath!}
                    mono
                    className="leading-snug"
                  />
                </CtxRow>
              )}
              {item.context.url && (
                <CtxRow icon={<Globe size={11} />} label="URL">
                  <CollapsibleText
                    value={item.context.url}
                    mono
                    className="leading-snug"
                  />
                </CtxRow>
              )}
              <CtxRow icon={<Clock size={11} />} label="Copied At">
                {new Date(item.context.capturedAt).toLocaleString()}
              </CtxRow>
            </dl>
          </div>
        )}

        <div className="flex items-center justify-between rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-2">
          <div className="flex items-center gap-2">
            <TypeBadge item={item} size="sm" />
            <div>
              <div className="text-[12px] font-medium text-[var(--mist)]">
                {typeLabel(item)}
              </div>
              <div className="text-[10px] text-[var(--subtle)]">
                Valid {typeLabel(item)} · {formatBytes(item.byteSize)}
                {item.detectedType === 'json' && ' · 42 items'}
              </div>
            </div>
          </div>
          {item.confidence != null && (
            <span className="rounded-full bg-[var(--lime-soft)] px-2 py-0.5 text-[10px] font-semibold text-[var(--lime)]">
              {item.confidence}% confidence
            </span>
          )}
        </div>
      </div>
    </aside>
  )
}

function PreviewTab({
  item,
  lines,
  lang,
  content,
  onEnlarge,
  fill,
  canJSON,
  actions,
  transformNote,
  transformError,
  transformBusy,
  onFormat,
  onMinify,
  onTransform,
  onTransformCopy,
  onResetTransform,
}: {
  item: ClipboardItem
  lines: string[]
  lang?: string
  content: string
  onEnlarge?: () => void
  fill?: boolean
  canJSON?: boolean
  actions?: TransformAction[]
  transformNote?: string | null
  transformError?: string | null
  transformBusy?: boolean
  onFormat?: () => void
  onMinify?: () => void
  onTransform?: (id: string) => void
  onTransformCopy?: (id: string) => void
  onResetTransform?: () => void
}) {
  if (item.detectedType === 'image' || item.detectedType === 'screenshot' || item.kind === 'image') {
    const fmt = (item.mimeType || '').split('/')[1]?.toUpperCase() || 'IMAGE'
    return (
      <div className={cn('m-3 flex min-h-0 flex-col space-y-2', fill && 'h-full')}>
        <button
          type="button"
          onClick={onEnlarge}
          className={cn(
            'group relative block w-full overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] text-left',
            fill && 'min-h-0 flex-1',
          )}
          aria-label="Enlarge image"
        >
          <ItemImage
            itemId={item.id}
            kind={item.imageKept === false ? 'thumb' : 'original'}
            revision={(item.thumbnailPath || '') + (item.blobPath || '')}
            fit="contain"
            className={cn('mx-auto w-full', fill ? 'h-full max-h-full' : 'max-h-[320px]')}
          />
          <span className="pointer-events-none absolute bottom-2 right-2 inline-flex items-center gap-1 rounded-md bg-[rgb(0_0_0_/0.55)] px-2 py-1 text-[10px] font-medium text-white opacity-0 transition-opacity group-hover:opacity-100">
            Enlarge
          </span>
        </button>
        <div className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-1 px-1 text-[11px] text-[var(--muted)]">
          {item.width && item.height ? (
            <span>
              {item.width} × {item.height}
            </span>
          ) : null}
          <span>{fmt}</span>
          {item.byteSize ? <span>{formatBytes(item.byteSize)}</span> : null}
          {item.imageKept === false ? (
            <span className="text-[var(--subtle)]">Original not kept (over 25 MB)</span>
          ) : null}
          <EnlargeImageButton onClick={() => onEnlarge?.()} className="ml-auto" />
        </div>
      </div>
    )
  }

  if (item.detectedType === 'color' && item.textContent) {
    return (
      <div className="m-3 space-y-3">
        <div
          className="h-32 rounded-[10px] border border-[var(--border)]"
          style={{ background: item.textContent }}
        />
        <p className="font-mono text-[13px] text-[var(--mist)]">
          {item.textContent}
        </p>
      </div>
    )
  }

  return (
    <div
      className={cn(
        'm-3 flex min-h-0 flex-col overflow-hidden rounded-[10px] border border-[var(--border)] bg-[var(--elevated)]',
        fill ? 'h-[calc(100%-1.5rem)]' : '',
      )}
    >
      <div className="flex shrink-0 flex-wrap items-center gap-1.5 border-b border-[var(--border)] px-3 py-1.5">
        <span className="rounded-md bg-[var(--elevated-2)] px-2 py-0.5 text-[10px] font-semibold uppercase text-[var(--muted)]">
          {lang || typeLabel(item)}
        </span>
        {canJSON && (
          <>
            <button
              type="button"
              disabled={transformBusy}
              onClick={onFormat}
              className="flex items-center gap-1 rounded-md px-2 py-0.5 text-[11px] text-[var(--muted)] hover:bg-[var(--elevated-2)] hover:text-[var(--mist)] disabled:opacity-50"
            >
              <Braces size={12} />
              Format
            </button>
            <button
              type="button"
              disabled={transformBusy}
              onClick={onMinify}
              className="rounded-md px-2 py-0.5 text-[11px] text-[var(--muted)] hover:bg-[var(--elevated-2)] hover:text-[var(--mist)] disabled:opacity-50"
            >
              Minify
            </button>
          </>
        )}
        {(actions || [])
          .filter((a) => a.group === 'generate' || a.group === 'view' || a.group === 'encode' || a.group === 'case')
          .slice(0, 8)
          .map((a) => (
            <button
              key={a.id}
              type="button"
              disabled={transformBusy}
              onClick={() => onTransform?.(a.id)}
              onDoubleClick={() => onTransformCopy?.(a.id)}
              title="Click to preview · double-click to copy"
              className="rounded-md px-2 py-0.5 text-[11px] text-[var(--muted)] hover:bg-[var(--elevated-2)] hover:text-[var(--mist)] disabled:opacity-50"
            >
              {a.label}
            </button>
          ))}
        {transformNote && (
          <button
            type="button"
            onClick={onResetTransform}
            className="ml-auto rounded-md px-2 py-0.5 text-[11px] text-[var(--lime)] hover:underline"
          >
            {transformNote} · reset
          </button>
        )}
      </div>
      {transformError && (
        <p className="shrink-0 border-b border-[var(--border)] px-3 py-1.5 text-[11px] text-[rgb(240_120_120)]">
          {transformError}
        </p>
      )}
      <pre className="min-h-0 flex-1 overflow-auto p-0 text-[12px] leading-5">
        <code className="block font-mono">
          {lines.map((line, i) => (
            <div key={i} className="flex hover:bg-[rgb(255_255_255_/0.02)]">
              <span className="w-10 shrink-0 select-none pr-3 text-right text-[11px] text-[var(--subtle)]">
                {i + 1}
              </span>
              <span
                className="min-w-0 flex-1 whitespace-pre-wrap break-all pr-3 text-[var(--mist)]"
                dangerouslySetInnerHTML={{
                  __html: highlightCode(line || ' ', lang),
                }}
              />
            </div>
          ))}
        </code>
      </pre>
      <span className="sr-only">{content}</span>
      {content.length >= 200_000 && (
        <p className="shrink-0 border-t border-[var(--border)] px-3 py-1.5 text-[11px] text-[var(--subtle)]">
          Showing the first 200 KB highlighted; full text still copies.
        </p>
      )}
    </div>
  )
}

function ContextTab({ item }: { item: ClipboardItem }) {
  const ctx = item.context
  if (!ctx) {
    return (
      <p className="px-4 py-6 text-[13px] text-[var(--subtle)]">
        No context captured for this item.
      </p>
    )
  }
  return (
    <div className="space-y-4 px-4 py-4">
      <section>
        <h3 className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-[var(--subtle)]">
          Context Information
        </h3>
        <div className="space-y-2 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] p-3 text-[12px]">
          <CtxRow icon={<AppWindow size={12} />} label="Application">
            <span className="inline-flex items-center gap-1.5">
              <AppIcon
                name={ctx.applicationName}
                iconKey={ctx.iconCacheKey || ctx.applicationId}
                size={18}
              />
              {ctx.applicationName}
            </span>
          </CtxRow>
          {ctx.documentTitle && (
            <CtxRow icon={<FileText size={12} />} label="Page Title">
              <CollapsibleText value={ctx.documentTitle} />
            </CtxRow>
          )}
          {ctx.url && (
            <CtxRow icon={<Globe size={12} />} label="URL">
              <CollapsibleText value={ctx.url} mono />
            </CtxRow>
          )}
          {ctx.domain && (
            <CtxRow icon={<Hash size={12} />} label="Domain">
              {ctx.domain}
            </CtxRow>
          )}
          {ctx.windowTitle && (
            <CtxRow icon={<AppWindow size={12} />} label="Window">
              <CollapsibleText value={ctx.windowTitle} />
            </CtxRow>
          )}
          {showContextFilePath(ctx.filePath, ctx.url) && (
            <CtxRow icon={<FileText size={12} />} label="File Path">
              <CollapsibleText value={ctx.filePath!} mono />
            </CtxRow>
          )}
          {ctx.processId != null && (
            <CtxRow icon={<Hash size={12} />} label="Process ID">
              {ctx.processId}
            </CtxRow>
          )}
          <CtxRow icon={<Clock size={12} />} label="Copied At">
            {new Date(ctx.capturedAt).toLocaleString()}
          </CtxRow>
        </div>
      </section>
      <section>
        <h3 className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-[var(--subtle)]">
          Content Information
        </h3>
        <div className="space-y-2 rounded-[10px] border border-[var(--border)] bg-[var(--elevated)] p-3 text-[12px]">
          <CtxRow label="Type">{typeLabel(item)}</CtxRow>
          <CtxRow label="Size">{formatBytes(item.byteSize)}</CtxRow>
          <CtxRow label="Format">{item.mimeType || '—'}</CtxRow>
          <CtxRow label="Sensitivity">{item.sensitivity}</CtxRow>
        </div>
      </section>
    </div>
  )
}

function ActionsTab({
  item,
  onRequestDelete,
}: {
  item: ClipboardItem
  onRequestDelete: () => void
}) {
  const [title, setTitle] = useState(item.title || '')
  const [msg, setMsg] = useState<string | null>(null)
  const isImage =
    item.kind === 'image' ||
    item.detectedType === 'image' ||
    item.detectedType === 'screenshot'
  const isFile = isFileItem(item)

  const flash = (m: string) => {
    setMsg(m)
    window.setTimeout(() => setMsg(null), 2000)
  }

  return (
    <div className="space-y-4 px-4 py-4">
      <section className="space-y-2">
        <h3 className="text-[11px] font-semibold uppercase tracking-wide text-[var(--subtle)]">
          Rename
        </h3>
        <div className="flex gap-2">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="min-w-0 flex-1 rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-2.5 py-1.5 text-[12px] text-[var(--mist)] outline-none focus:border-[var(--lime)]"
            placeholder="Title"
          />
          <button
            type="button"
            className="rounded-[8px] bg-[var(--elevated-2)] px-3 py-1.5 text-[12px] font-medium text-[var(--mist)] hover:bg-[var(--border)]"
            onClick={() =>
              void renameItem(item.id, title).then(() => flash('Renamed'))
            }
          >
            Save
          </button>
        </div>
      </section>

      {isFile && (
        <section className="space-y-2">
          <h3 className="text-[11px] font-semibold uppercase tracking-wide text-[var(--subtle)]">
            File
          </h3>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className="rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-1.5 text-[12px] text-[var(--mist)]"
              onClick={() =>
                void openFile(item.id)
                  .then(() => flash('Opened'))
                  .catch((e: Error) => flash(String(e.message || e)))
              }
            >
              Open
            </button>
            <button
              type="button"
              className="rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-1.5 text-[12px] text-[var(--mist)]"
              onClick={() =>
                void revealFile(item.id)
                  .then(() => flash('Revealed'))
                  .catch((e: Error) => flash(String(e.message || e)))
              }
            >
              Reveal
            </button>
          </div>
          <p className="text-[11px] text-[var(--subtle)]">
            Open with the default app, or reveal in Finder / Explorer.
          </p>
        </section>
      )}

      {isImage && (
        <section className="space-y-2">
          <h3 className="text-[11px] font-semibold uppercase tracking-wide text-[var(--subtle)]">
            Image
          </h3>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={item.imageKept === false}
              className="rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-1.5 text-[12px] text-[var(--mist)] disabled:opacity-40"
              onClick={() =>
                void saveImage(item.id)
                  .then(() => flash('Saved'))
                  .catch((e: Error) => flash(String(e.message || e)))
              }
            >
              Save…
            </button>
            <button
              type="button"
              disabled={item.imageKept === false}
              className="rounded-[8px] border border-[var(--border)] bg-[var(--elevated)] px-3 py-1.5 text-[12px] text-[var(--mist)] disabled:opacity-40"
              onClick={() =>
                void revealImage(item.id)
                  .then(() => flash('Revealed'))
                  .catch((e: Error) => flash(String(e.message || e)))
              }
            >
              Reveal
            </button>
          </div>
          {item.imageKept === false && (
            <p className="text-[11px] text-[var(--subtle)]">
              Original was not kept (over 25 MB). Save and Reveal are unavailable.
            </p>
          )}
        </section>
      )}

      {item.sensitivity === 'suspected' && (
        <section className="space-y-2">
          <h3 className="text-[11px] font-semibold uppercase tracking-wide text-[var(--subtle)]">
            Sensitive
          </h3>
          <button
            type="button"
            className="rounded-[8px] bg-[var(--lime)] px-3 py-1.5 text-[12px] font-semibold text-[var(--on-lime)]"
            onClick={() => void keepItem(item.id).then(() => flash('Kept'))}
          >
            Keep
          </button>
        </section>
      )}

      <section className="space-y-2">
        <h3 className="text-[11px] font-semibold uppercase tracking-wide text-[var(--subtle)]">
          Remove
        </h3>
        <button
          type="button"
          className="inline-flex items-center gap-1.5 rounded-[8px] border border-[rgb(220_70_70_/0.35)] bg-[rgb(220_70_70_/0.12)] px-3 py-1.5 text-[12px] font-medium text-[rgb(240_120_120)] hover:bg-[rgb(220_70_70_/0.2)]"
          onClick={onRequestDelete}
        >
          <Trash2 size={13} />
          Delete from history
        </button>
      </section>

      {msg && <p className="text-[12px] text-[var(--muted)]">{msg}</p>}

      <p className="text-[11px] text-[var(--subtle)]">
        Format / transform actions arrive in M4.
      </p>
    </div>
  )
}

function showContextFilePath(filePath?: string, url?: string): boolean {
  const path = filePath?.trim()
  if (!path) return false
  const lower = path.toLowerCase()
  if (lower.startsWith('http://') || lower.startsWith('https://')) return false
  if (url && path === url.trim()) return false
  return true
}

/** Long paths/URLs collapse to 2 lines with a show more / less control. */
function CollapsibleText({
  value,
  mono,
  className,
  collapseAt = 96,
}: {
  value: string
  mono?: boolean
  className?: string
  collapseAt?: number
}) {
  const [open, setOpen] = useState(false)
  const long = value.length > collapseAt || value.includes('\n')
  return (
    <div className="min-w-0">
      <span
        className={cn(
          'block break-all',
          mono && 'font-mono',
          !open && long && 'line-clamp-2',
          className,
        )}
      >
        {value}
      </span>
      {long && (
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="mt-0.5 inline-flex items-center gap-0.5 text-[10px] font-medium text-[var(--subtle)] hover:text-[var(--lime)]"
        >
          {open ? (
            <>
              Show less
              <ChevronUp size={11} />
            </>
          ) : (
            <>
              Show more
              <ChevronDown size={11} />
            </>
          )}
        </button>
      )}
    </div>
  )
}

function CtxRow({
  icon,
  label,
  children,
}: {
  icon?: ReactNode
  label: string
  children: ReactNode
}) {
  return (
    <div className="flex items-start gap-2">
      <dt className="flex w-[72px] shrink-0 items-center gap-1.5 pt-0.5 text-[var(--subtle)]">
        {icon}
        {label}
      </dt>
      <dd className="min-w-0 flex-1 overflow-hidden text-[var(--mist)]">{children}</dd>
    </div>
  )
}

function IconBtn({
  children,
  label,
  onClick,
  active,
}: {
  children: ReactNode
  label: string
  onClick?: () => void
  active?: boolean
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className={cn(
        'rounded-lg p-1.5 transition-colors duration-[120ms]',
        active
          ? 'text-[var(--lime)]'
          : 'text-[var(--subtle)] hover:bg-[var(--elevated)] hover:text-[var(--mist)]',
      )}
    >
      {children}
    </button>
  )
}
