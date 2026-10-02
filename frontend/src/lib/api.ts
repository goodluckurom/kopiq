import type { ClipboardItem, CopyContext, HistoryCounts } from './types'

type HistoryFilter = {
  kind?: string
  detectedType?: string
  appId?: string
  pinnedOnly?: boolean
  limit?: number
  offset?: number
}

type HistoryServiceNS = {
  List: (filter: HistoryFilter) => Promise<ClipboardItem[] | null>
  ListRecent: () => Promise<ClipboardItem[] | null>
  Get: (id: number) => Promise<ClipboardItem | null>
  Delete: (id: number) => Promise<void>
  Pin: (id: number, pinned: boolean) => Promise<void>
  CopyToClipboard: (id: number) => Promise<void>
  Counts?: () => Promise<HistoryCounts>
}

type SearchServiceNS = {
  Search: (query: string) => Promise<ClipboardItem[] | null>
}

function isWailsRuntime(): boolean {
  if (typeof window === 'undefined') return false
  const w = window as Window & { wails?: unknown; go?: unknown }
  return Boolean(w.wails || w.go)
}

function withTimeout<T>(p: Promise<T>, ms: number, label: string): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const t = window.setTimeout(() => reject(new Error(`${label} timed out`)), ms)
    p.then(
      (v) => {
        window.clearTimeout(t)
        resolve(v)
      },
      (e) => {
        window.clearTimeout(t)
        reject(e)
      },
    )
  })
}

/** Normalize Go DTO → frontend ClipboardItem (context alias, sensitivity). */
export function normalizeItem(raw: ClipboardItem & { lastContext?: CopyContext }): ClipboardItem {
  const context = raw.context ?? raw.lastContext
  const sensitivity =
    raw.sensitivity === 'normal' || !raw.sensitivity ? 'none' : raw.sensitivity
  return {
    ...raw,
    title: raw.title || titleFromItem(raw),
    sensitivity,
    context,
  }
}

function titleFromItem(item: ClipboardItem): string {
  if (item.title) return item.title
  if (item.detectedType === 'json') return 'JSON'
  if (item.detectedType === 'url') return item.textPreview?.slice(0, 60) || 'URL'
  if (item.detectedType === 'screenshot') return 'Screenshot'
  if (item.kind === 'image' || item.detectedType === 'image') return 'Image'
  if (
    item.detectedType === 'file' ||
    item.detectedType === 'files' ||
    item.kind === 'files'
  ) {
    const line = (item.textPreview || item.textContent || '').split('\n')[0]?.trim()
    if (line) {
      const base = line.split(/[/\\]/).filter(Boolean).pop()
      if (base) return base.slice(0, 72)
    }
  }
  const line = (item.textPreview || item.textContent || '').split('\n')[0]
  return line?.slice(0, 72) || 'Clipboard item'
}

let servicesPromise: Promise<{
  history: HistoryServiceNS | null
  search: SearchServiceNS | null
}> | null = null

async function getServices() {
  if (!servicesPromise) {
    servicesPromise = (async () => {
      if (!isWailsRuntime()) return { history: null, search: null }
      try {
        const mod = await import('../../bindings/kopiq/service')
        return {
          history: (mod.HistoryService ?? null) as HistoryServiceNS | null,
          search: (mod.SearchService ?? null) as SearchServiceNS | null,
        }
    } catch {
        return { history: null, search: null }
      }
    })()
  }
  return servicesPromise
}

function asList(items: unknown): ClipboardItem[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => normalizeItem(item as ClipboardItem))
}

export async function listItems(filter: HistoryFilter = {}): Promise<ClipboardItem[]> {
  const { history } = await getServices()

  try {
    if (
      history?.ListRecent &&
      !filter.kind &&
      !filter.detectedType &&
      !filter.appId &&
      !filter.pinnedOnly
    ) {
      const items = await withTimeout(history.ListRecent(), 4000, 'ListRecent')
      return asList(items)
    }
    if (history?.List) {
      const items = await withTimeout(
        history.List({ limit: 100, ...filter }),
        4000,
        'List',
      )
      return asList(items)
    }
  } catch (err) {
    console.error('listItems failed', err)
  }
  return []
}

export async function searchItems(query: string): Promise<ClipboardItem[]> {
  const { search } = await getServices()
  try {
    if (search?.Search) {
      const items = await withTimeout(search.Search(query), 4000, 'Search')
      return asList(items)
    }
  } catch (err) {
    console.error('searchItems failed', err)
  }
  if (!query.trim()) return listItems()
  return []
}

export async function getItem(id: number): Promise<ClipboardItem | null> {
  const { history } = await getServices()
    try {
    if (history?.Get) {
      const item = await withTimeout(history.Get(id), 4000, 'Get')
      return item ? normalizeItem(item) : null
    }
  } catch (err) {
    console.error('getItem failed', err)
  }
  return null
}

export async function fetchCounts(): Promise<HistoryCounts> {
  const empty: HistoryCounts = {
    total: 0,
    pinned: 0,
    snippets: 0,
    collections: 0,
    stacks: 0,
    byCategory: {},
    byApp: [],
  }
  const { history } = await getServices()
  try {
    if (history?.Counts) {
      const counts = await withTimeout(history.Counts(), 4000, 'Counts')
      return {
        total: counts?.total ?? 0,
        pinned: counts?.pinned ?? 0,
        snippets: counts?.snippets ?? 0,
        collections: counts?.collections ?? 0,
        stacks: counts?.stacks ?? 0,
        byCategory: counts?.byCategory ?? {},
        byApp: counts?.byApp ?? [],
      }
    }
  } catch (err) {
    console.error('fetchCounts failed', err)
  }
  return empty
}

export async function copyItem(id: number): Promise<void> {
  const { history } = await getServices()
  if (history?.CopyToClipboard) {
    await withTimeout(history.CopyToClipboard(id), 4000, 'CopyToClipboard')
      return
  }
  const item = await getItem(id)
  if (item?.textContent && navigator.clipboard) {
    await navigator.clipboard.writeText(item.textContent)
  }
}

export async function pasteItem(id: number): Promise<{ pasted?: boolean; copiedOnly?: boolean; message?: string }> {
  try {
    if (!isWailsRuntime()) {
      await copyItem(id)
      return { copiedOnly: true }
    }
    const mod = await import('../../bindings/kopiq/service/quickpasteservice')
    if (mod.PasteBack) {
      const result = await withTimeout(mod.PasteBack(id), 5000, 'PasteBack')
      return {
        pasted: Boolean(result?.pasted),
        copiedOnly: Boolean(result?.copiedOnly),
        message: result?.message || undefined,
      }
    }
  } catch (err) {
    console.error('pasteItem failed', err)
  }
  await copyItem(id)
  return { copiedOnly: true, message: 'Copied.' }
}

export async function pasteTransformed(
  itemId: number,
  transformId: string,
): Promise<{ pasted?: boolean; copiedOnly?: boolean; message?: string }> {
  try {
    if (!isWailsRuntime()) {
      const res = await applyTransform(itemId, transformId)
      if (res.text && navigator.clipboard) await navigator.clipboard.writeText(res.text)
      return { copiedOnly: true, message: res.label || 'Copied' }
    }
    const mod = await import('../../bindings/kopiq/service/quickpasteservice')
    if (mod.PasteTransformed) {
      const result = await withTimeout(
        mod.PasteTransformed(itemId, transformId),
        8000,
        'PasteTransformed',
      )
      return {
        pasted: Boolean(result?.pasted),
        copiedOnly: Boolean(result?.copiedOnly),
        message: result?.message || undefined,
      }
    }
    await applyTransformAndCopy(itemId, transformId)
    return { copiedOnly: true, message: 'Copied.' }
  } catch (err) {
    console.error('pasteTransformed failed', err)
    throw err
  }
}

export async function dismissQuickPaste(): Promise<void> {
  try {
    if (!isWailsRuntime()) return
    const mod = await import('../../bindings/kopiq/service/quickpasteservice')
    if (mod.Dismiss) await mod.Dismiss()
  } catch {
    // ignore
  }
}

export async function copyItemOnly(id: number): Promise<void> {
  await copyItem(id)
}

export async function pinItem(id: number, pinned: boolean): Promise<void> {
  const { history } = await getServices()
  if (history?.Pin) {
    await withTimeout(history.Pin(id, pinned), 4000, 'Pin')
    return
  }
  if (isWailsRuntime()) throw new Error('Pin is unavailable')
}

export async function deleteItem(id: number): Promise<void> {
  const { history } = await getServices()
  if (history?.Delete) {
    await withTimeout(history.Delete(id), 4000, 'Delete')
    return
  }
  if (isWailsRuntime()) throw new Error('Delete is unavailable')
}

export async function renameItem(id: number, title: string): Promise<void> {
  if (!isWailsRuntime()) return
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.Rename) await withTimeout(mod.Rename(id, title), 4000, 'Rename')
}

export async function saveImage(id: number): Promise<void> {
  if (!isWailsRuntime()) throw new Error('Save requires the desktop app')
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.SaveImage) await withTimeout(mod.SaveImage(id), 60000, 'SaveImage')
}

export async function revealImage(id: number): Promise<void> {
  if (!isWailsRuntime()) throw new Error('Reveal requires the desktop app')
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.RevealImage) await withTimeout(mod.RevealImage(id), 4000, 'RevealImage')
}

export async function openFile(id: number): Promise<void> {
  if (!isWailsRuntime()) throw new Error('Open requires the desktop app')
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.OpenFile) await withTimeout(mod.OpenFile(id), 4000, 'OpenFile')
  else throw new Error('OpenFile is unavailable')
}

export async function openContextFile(id: number): Promise<void> {
  if (!isWailsRuntime()) throw new Error('Open in editor requires the desktop app')
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.OpenContextFile) {
    await withTimeout(mod.OpenContextFile(id), 4000, 'OpenContextFile')
    return
  }
  throw new Error('OpenContextFile is unavailable — rebuild bindings')
}

export async function revealFile(id: number): Promise<void> {
  if (!isWailsRuntime()) throw new Error('Reveal requires the desktop app')
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.RevealFile) await withTimeout(mod.RevealFile(id), 4000, 'RevealFile')
  else throw new Error('RevealFile is unavailable')
}

export async function keepItem(id: number): Promise<void> {
  if (!isWailsRuntime()) return
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.Keep) await withTimeout(mod.Keep(id), 4000, 'Keep')
}

export async function clearHistory(): Promise<number> {
  if (!isWailsRuntime()) return 0
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.ClearHistory) return await withTimeout(mod.ClearHistory(), 8000, 'ClearHistory')
  return 0
}

export async function getSettings(): Promise<Record<string, string>> {
  try {
    if (!isWailsRuntime()) return {}
    const mod = await import('../../bindings/kopiq/service/settingsservice')
    if (!mod.Get) return {}
    const raw = await withTimeout(mod.Get(), 4000, 'Settings.Get')
    const out: Record<string, string> = {}
    if (raw && typeof raw === 'object') {
      for (const [k, v] of Object.entries(raw)) {
        if (typeof v === 'string') out[k] = v
      }
    }
    return out
  } catch {
    return {}
  }
}

export async function setSetting(key: string, value: string): Promise<void> {
  if (!isWailsRuntime()) return
  const mod = await import('../../bindings/kopiq/service/settingsservice')
  if (mod.Set) await withTimeout(mod.Set(key, value), 4000, 'Settings.Set')
}

export type StorageBreakdown = {
  textBytes: number
  imageBytes: number
  fileBytes: number
  databaseBytes: number
  totalBytes: number
  itemCount: number
  pinnedCount: number
}

export async function getStorageStats(): Promise<StorageBreakdown | null> {
  try {
    if (!isWailsRuntime()) return null
    const mod = await import('../../bindings/kopiq/service/settingsservice')
    if (!mod.StorageStats) return null
    const raw = await withTimeout(mod.StorageStats(), 5000, 'StorageStats')
    if (!raw || typeof raw !== 'object') return null
    return {
      textBytes: Number(raw.textBytes) || 0,
      imageBytes: Number(raw.imageBytes) || 0,
      fileBytes: Number(raw.fileBytes) || 0,
      databaseBytes: Number(raw.databaseBytes) || 0,
      totalBytes: Number(raw.totalBytes) || 0,
      itemCount: Number(raw.itemCount) || 0,
      pinnedCount: Number(raw.pinnedCount) || 0,
    }
  } catch {
    return null
  }
}

export async function cleanUpHistory(): Promise<{ removed: number; bytesFreed: number }> {
  if (!isWailsRuntime()) return { removed: 0, bytesFreed: 0 }
  const mod = await import('../../bindings/kopiq/service/settingsservice')
  if (!mod.CleanUp) return { removed: 0, bytesFreed: 0 }
  const raw = await withTimeout(mod.CleanUp(), 60000, 'CleanUp')
  return {
    removed: Number((raw as { removed?: number }).removed) || 0,
    bytesFreed: Number((raw as { bytesFreed?: number }).bytesFreed) || 0,
  }
}

export async function clearDeepFields(): Promise<number> {
  if (!isWailsRuntime()) return 0
  const mod = await import('../../bindings/kopiq/service/settingsservice')
  if (!mod.ClearDeepFields) return 0
  return Number(await withTimeout(mod.ClearDeepFields(), 15000, 'ClearDeepFields')) || 0
}

export type ExclusionRow = {
  id: number
  applicationId: string
  applicationName: string
  source: string
  enabled: boolean
  iconCacheKey?: string
}

export async function listExclusions(): Promise<ExclusionRow[]> {
  try {
    if (!isWailsRuntime()) return []
    const mod = await import('../../bindings/kopiq/service/historyservice')
    if (!mod.ListExclusions) return []
    const rows = await withTimeout(mod.ListExclusions(), 8000, 'ListExclusions')
    if (!Array.isArray(rows)) return []
    return rows.map((r) => ({
      id: Number(r.id),
      applicationId: r.applicationId || '',
      applicationName: r.applicationName || '',
      source: r.source || '',
      enabled: Boolean(r.enabled),
      iconCacheKey: r.iconCacheKey || r.applicationId || '',
    }))
  } catch {
    return []
  }
}

export async function setExclusionEnabled(id: number, enabled: boolean): Promise<void> {
  if (!isWailsRuntime()) return
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (mod.SetExclusionEnabled) {
    await withTimeout(mod.SetExclusionEnabled(id, enabled), 4000, 'SetExclusionEnabled')
  }
}

export async function addExclusion(
  applicationId: string,
  applicationName: string,
): Promise<void> {
  if (!isWailsRuntime()) throw new Error('Exclusions require the desktop app')
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (!mod.AddExclusion) throw new Error('AddExclusion unavailable')
  await withTimeout(
    mod.AddExclusion(applicationId, applicationName),
    8000,
    'AddExclusion',
  )
}

export async function deleteExclusion(id: number): Promise<void> {
  if (!isWailsRuntime()) throw new Error('Exclusions require the desktop app')
  const mod = await import('../../bindings/kopiq/service/historyservice')
  const del = (mod as { DeleteExclusion?: (id: number) => Promise<void> }).DeleteExclusion
  if (!del) throw new Error('DeleteExclusion unavailable')
  await withTimeout(del(id), 4000, 'DeleteExclusion')
}

export type ResolvedApp = {
  applicationId: string
  applicationName: string
  iconCacheKey?: string
  found: boolean
}

/** Resolve display name + cache icon for a bundle id / app path / app name. */
export async function resolveApp(query: string): Promise<ResolvedApp | null> {
  const id = query.trim()
  if (!id || !isWailsRuntime()) return null
  try {
    const mod = await import('../../bindings/kopiq/service/historyservice')
    if (typeof (mod as { ResolveApp?: (id: string) => Promise<ResolvedApp> }).ResolveApp === 'function') {
      const info = await withTimeout(
        (mod as { ResolveApp: (id: string) => Promise<ResolvedApp> }).ResolveApp(id),
        8000,
        'ResolveApp',
      )
      return {
        applicationId: info?.applicationId || '',
        applicationName: info?.applicationName || '',
        iconCacheKey: info?.iconCacheKey || info?.applicationId || id,
        found: Boolean(info?.found),
      }
    }
    if (mod.AppIconDataURL) {
      await withTimeout(mod.AppIconDataURL(id), 8000, 'AppIconDataURL')
    }
    return { applicationId: id, applicationName: '', iconCacheKey: id, found: false }
  } catch {
    return { applicationId: id, applicationName: '', iconCacheKey: id, found: false }
  }
}

export type FrontmostApp = {
  applicationId: string
  applicationName: string
}

async function callSettings<T>(method: string, ...args: unknown[]): Promise<T> {
  const { Call } = await import('@wailsio/runtime')
  return withTimeout(
    Call.ByName(`kopiq/service.SettingsService.${method}`, ...args) as Promise<T>,
    5000,
    method,
  )
}

export async function needsOnboarding(): Promise<boolean> {
  // Vite dev / HMR: always re-run onboarding on refresh.
  if (import.meta.env.DEV) return true
  try {
    if (!isWailsRuntime()) return false
    // Prefer generated binding when present.
    const mod = await import('../../bindings/kopiq/service/settingsservice')
    if (typeof (mod as { NeedsOnboarding?: () => Promise<boolean> }).NeedsOnboarding === 'function') {
      return Boolean(
        await withTimeout(
          (mod as { NeedsOnboarding: () => Promise<boolean> }).NeedsOnboarding(),
          4000,
          'NeedsOnboarding',
        ),
      )
    }
    return Boolean(await callSettings<boolean>('NeedsOnboarding'))
  } catch {
    const s = await getSettings()
    return s['onboarding.completed'] !== 'true' && s['onboarding.completed'] !== '1'
  }
}

export async function completeOnboarding(): Promise<void> {
  if (!isWailsRuntime()) return
  try {
    const mod = await import('../../bindings/kopiq/service/settingsservice')
    if (typeof (mod as { CompleteOnboarding?: () => Promise<void> }).CompleteOnboarding === 'function') {
      await withTimeout(
        (mod as { CompleteOnboarding: () => Promise<void> }).CompleteOnboarding(),
        4000,
        'CompleteOnboarding',
      )
      return
    }
  } catch {
    // fall through
  }
  await callSettings<void>('CompleteOnboarding')
}

export async function accessibilityGranted(): Promise<boolean> {
  try {
    if (!isWailsRuntime()) return false
    return Boolean(await callSettings<boolean>('AccessibilityGranted'))
  } catch {
    return false
  }
}

export async function openAccessibilitySettings(): Promise<void> {
  if (!isWailsRuntime()) return
  await callSettings<void>('OpenAccessibilitySettings')
}

export async function getLaunchFrontmost(): Promise<FrontmostApp | null> {
  try {
    if (!isWailsRuntime()) return null
    const raw = await callSettings<{
      applicationId?: string
      applicationName?: string
      ApplicationID?: string
      ApplicationName?: string
    }>('LaunchFrontmost')
    if (!raw) return null
    return {
      applicationId: raw.applicationId || raw.ApplicationID || '',
      applicationName: raw.applicationName || raw.ApplicationName || '',
    }
  } catch {
    return null
  }
}

export async function excludeLaunchFrontmost(): Promise<void> {
  if (!isWailsRuntime()) return
  await callSettings<void>('ExcludeLaunchFrontmost')
}

async function organizeMod() {
  return import('../../bindings/kopiq/service/organizeservice')
}

export async function listCollections(): Promise<import('./types').Collection[]> {
  if (!isWailsRuntime()) return []
  const mod = await organizeMod()
  const rows = (await withTimeout(mod.ListCollections(), 4000, 'ListCollections')) || []
  return rows as import('./types').Collection[]
}

export async function createCollection(name: string): Promise<import('./types').Collection> {
  const mod = await organizeMod()
  return (await withTimeout(mod.CreateCollection(name), 4000, 'CreateCollection')) as import('./types').Collection
}

export async function renameCollection(id: number, name: string): Promise<void> {
  const mod = await organizeMod()
  await withTimeout(mod.RenameCollection(id, name), 4000, 'RenameCollection')
}

export async function deleteCollection(id: number): Promise<void> {
  const mod = await organizeMod()
  await withTimeout(mod.DeleteCollection(id), 4000, 'DeleteCollection')
}

export async function addToCollection(collectionId: number, itemId: number): Promise<void> {
  const mod = await organizeMod()
  await withTimeout(mod.AddToCollection(collectionId, itemId), 4000, 'AddToCollection')
}

export async function listCollectionItems(
  collectionId: number,
): Promise<import('./types').ClipboardItem[]> {
  if (!isWailsRuntime()) return []
  const mod = await organizeMod()
  const rows =
    (await withTimeout(mod.ListCollectionItems(collectionId), 5000, 'ListCollectionItems')) || []
  return (rows as import('./types').ClipboardItem[]).map((r) => normalizeItem(r))
}

export async function listSnippets(): Promise<import('./types').Snippet[]> {
  if (!isWailsRuntime()) return []
  const mod = await organizeMod()
  return ((await withTimeout(mod.ListSnippets(), 4000, 'ListSnippets')) || []) as import('./types').Snippet[]
}

export async function saveSnippetFromItem(
  itemId: number,
  title: string,
  alias: string,
): Promise<import('./types').Snippet> {
  const mod = await organizeMod()
  return (await withTimeout(
    mod.SaveSnippetFromItem(itemId, title, alias),
    4000,
    'SaveSnippetFromItem',
  )) as import('./types').Snippet
}

export async function deleteSnippet(id: number): Promise<void> {
  const mod = await organizeMod()
  await withTimeout(mod.DeleteSnippet(id), 4000, 'DeleteSnippet')
}

export async function setSnippetAlias(id: number, alias: string): Promise<void> {
  const mod = await organizeMod()
  await withTimeout(mod.SetSnippetAlias(id, alias), 4000, 'SetSnippetAlias')
}

export type TransformAction = { id: string; label: string; group: string }
export type TransformResult = {
  id: string
  label: string
  text?: string
  error?: string
  kind: string
}

async function transformMod() {
  return import('../../bindings/kopiq/service/transformservice')
}

export async function listTransformActions(itemId: number): Promise<TransformAction[]> {
  if (!isWailsRuntime()) return []
  try {
    const mod = await transformMod()
    const rows = (await withTimeout(mod.ListActions(itemId), 4000, 'ListActions')) || []
    return rows as TransformAction[]
  } catch {
    return []
  }
}

export async function applyTransform(
  itemId: number,
  transformId: string,
): Promise<TransformResult> {
  const mod = await transformMod()
  return (await withTimeout(
    mod.Apply(itemId, transformId),
    8000,
    'Apply',
  )) as TransformResult
}

export async function applyTransformAndCopy(
  itemId: number,
  transformId: string,
): Promise<TransformResult> {
  if (!isWailsRuntime()) {
    return applyTransform(itemId, transformId)
  }
  const mod = await transformMod()
  return (await withTimeout(
    mod.ApplyAndCopy(itemId, transformId),
    8000,
    'ApplyAndCopy',
  )) as TransformResult
}

export async function multiCopy(
  ids: number[],
  shape: 'plain' | 'lines' | 'comma' | 'csv' | 'json' | 'markdown',
): Promise<{ text: string; imagesOmitted: number; itemsUsed: number; note?: string }> {
  if (!isWailsRuntime()) {
    throw new Error('Multi-copy requires the desktop app')
  }
  const mod = await import('../../bindings/kopiq/service/historyservice')
  if (!mod.MultiCopy) throw new Error('MultiCopy unavailable — rebuild bindings')
  return (await withTimeout(mod.MultiCopy(ids, shape), 8000, 'MultiCopy')) as {
    text: string
    imagesOmitted: number
    itemsUsed: number
    note?: string
  }
}

export async function startStack(ids: number[]): Promise<{ armed: boolean; label: string }> {
  if (!isWailsRuntime()) throw new Error('Stacks require the desktop app')
  const mod = await import('../../bindings/kopiq/service/stackservice')
  return (await withTimeout(mod.StartFromItems(ids), 4000, 'StartFromItems')) as {
    armed: boolean
    label: string
  }
}

export async function listStacks(): Promise<
  { id: number; name: string; createdAt: number; stepCount: number }[]
> {
  if (!isWailsRuntime()) return []
  const mod = await import('../../bindings/kopiq/service/stackservice')
  const rows = (await withTimeout(mod.List(), 4000, 'ListStacks')) || []
  return rows as { id: number; name: string; createdAt: number; stepCount: number }[]
}

export async function startSavedStack(
  id: number,
): Promise<{ armed: boolean; label: string }> {
  if (!isWailsRuntime()) throw new Error('Stacks require the desktop app')
  const mod = await import('../../bindings/kopiq/service/stackservice')
  return (await withTimeout(mod.Start(id), 4000, 'StartStack')) as {
    armed: boolean
    label: string
  }
}

export async function deleteStack(id: number): Promise<void> {
  if (!isWailsRuntime()) return
  const mod = await import('../../bindings/kopiq/service/stackservice')
  await mod.Delete(id)
}

export async function stopStack(): Promise<void> {
  if (!isWailsRuntime()) return
  const mod = await import('../../bindings/kopiq/service/stackservice')
  await mod.Stop()
}

export async function stackStatus(): Promise<{
  armed: boolean
  index: number
  total: number
  label: string
  nextPreview?: string
}> {
  if (!isWailsRuntime()) {
    return { armed: false, index: 0, total: 0, label: '' }
  }
  const mod = await import('../../bindings/kopiq/service/stackservice')
  return (await withTimeout(mod.Status(), 2000, 'StackStatus')) as {
    armed: boolean
    index: number
    total: number
    label: string
    nextPreview?: string
  }
}
