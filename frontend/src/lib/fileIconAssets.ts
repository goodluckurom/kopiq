/**
 * Real file-type logos from vscode-icons (@iconify-json/vscode-icons).
 * Uses a generated extension→icon map (~1400+) plus category fallbacks.
 */
import vsPack from '@iconify-json/vscode-icons/icons.json'
import generatedMap from './generatedFileIconMap.json'
import {
  extensionFromPath,
  pathFromFileItem,
  type FileCategory,
  fileVisualFromExtension,
} from './fileIcons'

type IconifyIcon = {
  body: string
  width?: number | string
  height?: number | string
}

type IconifyPack = {
  icons: Record<string, IconifyIcon>
  aliases?: Record<string, { parent: string }>
  width?: number
  height?: number
}

const pack = vsPack as IconifyPack
const EXT_TO_ICON = generatedMap as Record<string, string>

function hasIcon(name: string): boolean {
  return Boolean(pack.icons[name] || pack.aliases?.[name])
}

function getIcon(name: string): IconifyIcon | null {
  if (pack.icons[name]) return pack.icons[name]
  const parent = pack.aliases?.[name]?.parent
  if (parent && pack.icons[parent]) return pack.icons[parent]
  return null
}

/** Hand-picked overrides that must win over the generated map. */
const OVERRIDE: Record<string, string> = {
  go: 'file-type-go-gopher',
  tsx: 'file-type-reactts',
  ts: 'file-type-typescript-official',
  js: 'file-type-js-official',
  jsx: 'file-type-reactjs',
  py: 'file-type-python',
  rs: 'file-type-rust',
  json: 'file-type-json-official',
  yaml: 'file-type-yaml-official',
  yml: 'file-type-yaml-official',
  pdf: 'file-type-pdf2',
  zip: 'file-type-zip2',
}

const CATEGORY_FALLBACK: Record<FileCategory, string[]> = {
  image: ['file-type-image'],
  audio: ['file-type-audio'],
  video: ['file-type-video'],
  document: ['file-type-text', 'file-type-word'],
  spreadsheet: ['file-type-excel', 'file-type-csv'],
  presentation: ['file-type-powerpoint'],
  archive: ['file-type-zip2', 'file-type-zip'],
  code: ['file-type-binary', 'file-type-text'],
  markup: ['file-type-html', 'file-type-xml'],
  data: ['file-type-json', 'file-type-yaml'],
  config: ['file-type-config'],
  font: ['file-type-font'],
  ebook: ['file-type-epub', 'file-type-text'],
  app: ['file-type-package', 'file-type-binary'],
  disk: ['file-type-package', 'file-type-binary'],
  installer: ['file-type-package', 'file-type-binary'],
  design: ['file-type-image'],
  '3d': ['file-type-gltf', 'file-type-binary'],
  database: ['file-type-db', 'file-type-sql'],
  cert: ['file-type-cert', 'file-type-key'],
  mail: ['file-type-mailing', 'file-type-text'],
  calendar: ['file-type-text'],
  script: ['file-type-shell'],
  binary: ['file-type-binary'],
  other: ['file-type-text'],
}

const svgCache = new Map<string, string>()

function firstExisting(candidates: string[]): string | null {
  for (const name of candidates) {
    if (hasIcon(name)) return name
  }
  return null
}

/** Resolve vscode-icons icon id for a file extension. */
export function resolveFileIconName(ext: string): string {
  const e = ext.toLowerCase().replace(/^\./, '')
  if (!e) return firstExisting(['file-type-text']) || 'file-type-text'

  if (OVERRIDE[e] && hasIcon(OVERRIDE[e])) return OVERRIDE[e]

  const mapped = EXT_TO_ICON[e]
  if (mapped && hasIcon(mapped)) return mapped

  const direct = firstExisting([
    `file-type-${e}`,
    `file-type-${e}2`,
    `file-type-${e}-official`,
    `file-type-light-${e}`,
  ])
  if (direct) return direct

  // Compound: react.ts style already handled by extensionFromPath;
  // try stripping digits (e.g. php3 → php)
  const stripped = e.replace(/\d+$/, '')
  if (stripped && stripped !== e) {
    const again = EXT_TO_ICON[stripped]
    if (again && hasIcon(again)) return again
    const d2 = firstExisting([`file-type-${stripped}`, `file-type-${stripped}2`])
    if (d2) return d2
  }

  const { category } = fileVisualFromExtension(e)
  const cat = firstExisting(CATEGORY_FALLBACK[category] || CATEGORY_FALLBACK.other)
  return cat || 'file-type-text'
}

export function svgDataUrlForIcon(iconName: string): string {
  const cached = svgCache.get(iconName)
  if (cached) return cached

  const icon = getIcon(iconName) || getIcon('file-type-text')
  if (!icon) {
    const empty =
      'data:image/svg+xml,' +
      encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16" rx="2" fill="#94a3b8"/></svg>',
      )
    svgCache.set(iconName, empty)
    return empty
  }

  const w = Number(icon.width ?? pack.width ?? 32)
  const h = Number(icon.height ?? pack.height ?? 32)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${w} ${h}" width="${w}" height="${h}" preserveAspectRatio="xMidYMid meet">${icon.body}</svg>`
  const url = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
  svgCache.set(iconName, url)
  return url
}

export function fileIconDataUrlFromExtension(ext: string): string {
  return svgDataUrlForIcon(resolveFileIconName(ext))
}

export function fileIconDataUrlFromPath(path: string): string {
  return fileIconDataUrlFromExtension(extensionFromPath(path))
}

export function fileIconDataUrlFromItem(item: {
  title?: string
  textPreview?: string
  textContent?: string
}): string {
  return fileIconDataUrlFromPath(pathFromFileItem(item))
}

/** How many extensions have an explicit logo mapping. */
export function mappedExtensionCount(): number {
  return Object.keys(EXT_TO_ICON).length
}
