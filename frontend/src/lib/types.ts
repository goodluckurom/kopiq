export type CopyContext = {
  applicationName: string
  applicationId: string
  iconCacheKey?: string
  windowTitle?: string
  documentTitle?: string
  url?: string
  domain?: string
  fileName?: string
  filePath?: string
  processId?: number
  capturedAt: number
}

export type ClipboardItem = {
  id: number
  kind: string
  detectedType: string
  language?: string
  title: string
  textPreview: string
  textContent?: string
  mimeType?: string
  blobPath?: string
  thumbnailPath?: string
  byteSize: number
  width?: number
  height?: number
  imageKept?: boolean
  isPinned: boolean
  sensitivity: string
  copyCount: number
  createdAt: number
  lastCopiedAt: number
  context?: CopyContext
  tags?: string[]
  confidence?: number
  metadataJson?: string
}

export type CategoryId =
  | 'all'
  | 'pinned'
  | 'images'
  | 'code'
  | 'links'
  | 'text'
  | 'files'
  | 'snippets'
  | 'collections'
  | 'stacks'

export type NavFilter = {
  category: CategoryId
  sourceApp?: string
  tag?: string
  collectionId?: number
  /** App-lens type toggle (KQ-007). */
  lensType?: 'text' | 'images' | 'links' | 'code' | 'files'
  /** App-lens period toggle (KQ-007). */
  lensPeriod?: 'today' | 'week'
}

export type Collection = {
  id: number
  name: string
  createdAt: number
  sortOrder: number
  itemCount: number
}

export type Snippet = {
  id: number
  sourceItemId?: number
  title: string
  body: string
  language?: string
  createdAt: number
  updatedAt: number
  aliases?: string[]
}

export type ClipboardStack = {
  id: number
  name: string
  createdAt: number
  stepCount: number
}

export type AppCount = {
  name: string
  appId: string
  count: number
  iconCacheKey?: string
}

export type HistoryCounts = {
  total: number
  pinned: number
  snippets?: number
  collections?: number
  stacks?: number
  byCategory: Record<string, number>
  byApp: AppCount[]
}
