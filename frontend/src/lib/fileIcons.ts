/**
 * File-extension visuals for clipboard file items.
 * Maps hundreds of common extensions → Lucide icon + tint.
 * Unknown extensions fall back to a generic file look.
 */
import type { LucideIcon } from 'lucide-react'
import {
  Archive,
  Atom,
  Binary,
  BookOpen,
  Box,
  Braces,
  Brush,
  Calendar,
  Camera,
  Clapperboard,
  Code2,
  Coffee,
  Component,
  Container,
  Cpu,
  Database,
  Disc3,
  FileArchive,
  FileCode,
  FileCog,
  FileImage,
  FileJson,
  FileKey,
  FileLock,
  FileMusic,
  FilePen,
  FileSpreadsheet,
  FileText,
  FileType,
  FileType2,
  FileVideo,
  Film,
  FolderArchive,
  Gamepad2,
  Globe,
  HardDrive,
  Headphones,
  Hexagon,
  Image,
  KeyRound,
  Layers,
  Layout,
  Mail,
  Map,
  Mic,
  MonitorSmartphone,
  Music,
  Notebook,
  Package,
  Palette,
  PenTool,
  Presentation,
  Printer,
  ScrollText,
  Settings2,
  Sheet,
  Shield,
  Smartphone,
  Table,
  Terminal,
  Type,
  Variable,
  Workflow,
  File,
  AppWindow,
} from 'lucide-react'
import generatedFileIconMap from './generatedFileIconMap.json'

export type FileVisual = {
  icon: LucideIcon
  /** Icon stroke / fill tint */
  color: string
  /** Soft chip background */
  bg: string
  /** Short badge label (usually uppercase extension) */
  label: string
  category: FileCategory
}

export type FileCategory =
  | 'image'
  | 'audio'
  | 'video'
  | 'document'
  | 'spreadsheet'
  | 'presentation'
  | 'archive'
  | 'code'
  | 'markup'
  | 'data'
  | 'config'
  | 'font'
  | 'ebook'
  | 'app'
  | 'disk'
  | 'installer'
  | 'design'
  | '3d'
  | 'database'
  | 'cert'
  | 'mail'
  | 'calendar'
  | 'script'
  | 'binary'
  | 'other'

type CategoryStyle = {
  icon: LucideIcon
  color: string
  bg: string
}

/** Light-mode friendly tints (work on dark too via opacity). */
const CATEGORY: Record<FileCategory, CategoryStyle> = {
  image: { icon: FileImage, color: '#0d9488', bg: 'rgb(13 148 136 / 0.12)' },
  audio: { icon: FileMusic, color: '#7c3aed', bg: 'rgb(124 58 237 / 0.12)' },
  video: { icon: FileVideo, color: '#db2777', bg: 'rgb(219 39 119 / 0.12)' },
  document: { icon: FileText, color: '#2563eb', bg: 'rgb(37 99 235 / 0.12)' },
  spreadsheet: { icon: FileSpreadsheet, color: '#16a34a', bg: 'rgb(22 163 74 / 0.12)' },
  presentation: { icon: Presentation, color: '#ea580c', bg: 'rgb(234 88 12 / 0.12)' },
  archive: { icon: FileArchive, color: '#a16207', bg: 'rgb(161 98 7 / 0.14)' },
  code: { icon: FileCode, color: '#4f46e5', bg: 'rgb(79 70 229 / 0.12)' },
  markup: { icon: Code2, color: '#e11d48', bg: 'rgb(225 29 72 / 0.12)' },
  data: { icon: FileJson, color: '#0891b2', bg: 'rgb(8 145 178 / 0.12)' },
  config: { icon: FileCog, color: '#64748b', bg: 'rgb(100 116 139 / 0.14)' },
  font: { icon: Type, color: '#9333ea', bg: 'rgb(147 51 234 / 0.12)' },
  ebook: { icon: BookOpen, color: '#b45309', bg: 'rgb(180 83 9 / 0.12)' },
  app: { icon: Package, color: '#0f766e', bg: 'rgb(15 118 110 / 0.12)' },
  disk: { icon: HardDrive, color: '#475569', bg: 'rgb(71 85 105 / 0.14)' },
  installer: { icon: Box, color: '#c2410c', bg: 'rgb(194 65 12 / 0.12)' },
  design: { icon: PenTool, color: '#d946ef', bg: 'rgb(217 70 239 / 0.12)' },
  '3d': { icon: Box, color: '#6366f1', bg: 'rgb(99 102 241 / 0.12)' },
  database: { icon: Database, color: '#0284c7', bg: 'rgb(2 132 199 / 0.12)' },
  cert: { icon: Shield, color: '#059669', bg: 'rgb(5 150 105 / 0.12)' },
  mail: { icon: Mail, color: '#2563eb', bg: 'rgb(37 99 235 / 0.12)' },
  calendar: { icon: Calendar, color: '#dc2626', bg: 'rgb(220 38 38 / 0.12)' },
  script: { icon: Terminal, color: '#334155', bg: 'rgb(51 65 85 / 0.12)' },
  binary: { icon: Binary, color: '#57534e', bg: 'rgb(87 83 78 / 0.14)' },
  other: { icon: File, color: '#64748b', bg: 'rgb(100 116 139 / 0.12)' },
}

/** Per-extension icon overrides (otherwise category default). */
const ICON_OVERRIDE: Record<string, LucideIcon> = {
  // images
  png: Image,
  jpg: Camera,
  jpeg: Camera,
  gif: Image,
  webp: Image,
  svg: Palette,
  ico: Image,
  heic: Camera,
  raw: Camera,
  // audio
  mp3: Music,
  wav: Headphones,
  flac: Music,
  aac: Headphones,
  m4a: Music,
  ogg: Mic,
  // video
  mp4: Film,
  mov: Clapperboard,
  mkv: Film,
  webm: Film,
  avi: Film,
  // docs
  pdf: FileType,
  doc: FileText,
  docx: FileText,
  rtf: FileType2,
  txt: ScrollText,
  md: FilePen,
  markdown: FilePen,
  // sheets / slides
  xls: Sheet,
  xlsx: Sheet,
  csv: Table,
  tsv: Table,
  ppt: Presentation,
  pptx: Presentation,
  key: Presentation,
  numbers: Sheet,
  pages: FileText,
  // archives
  zip: Archive,
  rar: FolderArchive,
  '7z': Archive,
  tar: Archive,
  gz: Archive,
  tgz: Archive,
  bz2: Archive,
  xz: Archive,
  // apps / installers
  dmg: Disc3,
  iso: Disc3,
  img: HardDrive,
  exe: Cpu,
  msi: Package,
  app: AppWindow,
  apk: Smartphone,
  ipa: MonitorSmartphone,
  deb: Package,
  rpm: Package,
  pkg: Package,
  appx: Package,
  msix: Package,
  // code / langs
  ts: FileCode,
  tsx: Component,
  js: FileCode,
  jsx: Component,
  mjs: FileCode,
  cjs: FileCode,
  py: Atom,
  go: Hexagon,
  rs: Settings2,
  java: Coffee,
  kt: Hexagon,
  kts: Hexagon,
  swift: Hexagon,
  rb: Hexagon,
  php: Globe,
  cs: FileCode,
  cpp: FileCode,
  cxx: FileCode,
  cc: FileCode,
  c: FileCode,
  h: FileCode,
  hpp: FileCode,
  m: FileCode,
  mm: FileCode,
  scala: FileCode,
  clj: Braces,
  cljs: Braces,
  lua: FileCode,
  r: Variable,
  jl: Variable,
  dart: Hexagon,
  zig: FileCode,
  nim: FileCode,
  hs: FileCode,
  elm: FileCode,
  ex: FileCode,
  exs: FileCode,
  erl: FileCode,
  vue: Component,
  svelte: Component,
  astro: Globe,
  // markup / style
  html: Globe,
  htm: Globe,
  css: Brush,
  scss: Brush,
  sass: Brush,
  less: Brush,
  styl: Brush,
  xml: Code2,
  xhtml: Globe,
  // data
  json: Braces,
  jsonc: Braces,
  json5: Braces,
  yaml: FileCog,
  yml: FileCog,
  toml: FileCog,
  ini: Settings2,
  env: FileLock,
  sql: Database,
  graphql: Hexagon,
  gql: Hexagon,
  proto: Layers,
  parquet: Database,
  avro: Database,
  // config / devops
  dockerfile: Container,
  containerfile: Container,
  tf: Workflow,
  hcl: Workflow,
  nix: Settings2,
  makefile: Terminal,
  cmake: Settings2,
  gradle: Settings2,
  // fonts
  ttf: Type,
  otf: Type,
  woff: Type,
  woff2: Type,
  eot: Type,
  // design
  fig: PenTool,
  sketch: PenTool,
  xd: PenTool,
  psd: Palette,
  ai: PenTool,
  eps: PenTool,
  indd: Layout,
  blend: Box,
  // 3d
  obj: Box,
  fbx: Box,
  gltf: Box,
  glb: Box,
  stl: Box,
  '3ds': Box,
  dae: Box,
  // ebooks
  epub: BookOpen,
  mobi: BookOpen,
  azw: BookOpen,
  azw3: BookOpen,
  fb2: Notebook,
  // cert / keys
  pem: FileKey,
  crt: Shield,
  cer: Shield,
  p12: KeyRound,
  pfx: KeyRound,
  // .key is Keynote on macOS; PEM-style keys use .pem / .pub
  pub: FileKey,
  // mail / calendar
  eml: Mail,
  msg: Mail,
  ics: Calendar,
  vcf: Mail,
  // maps / geo
  geojson: Map,
  kml: Map,
  gpx: Map,
  // games / misc
  unity: Gamepad2,
  godot: Gamepad2,
  // print
  print: Printer,
}

/**
 * Extension → category. Many extensions share a category icon;
 * ICON_OVERRIDE picks a more specific glyph when present.
 */
const EXT_CATEGORY: Record<string, FileCategory> = {}

function add(cat: FileCategory, exts: string[]) {
  for (const e of exts) EXT_CATEGORY[e.toLowerCase()] = cat
}

// ── Images (raster + vector + raw) ──────────────────────────────────────────
add('image', [
  'png', 'jpg', 'jpeg', 'jpe', 'jfif', 'gif', 'webp', 'bmp', 'dib', 'tif', 'tiff',
  'ico', 'icns', 'svg', 'svgz', 'heic', 'heif', 'avif', 'apng', 'jxl', 'jp2', 'j2k',
  'raw', 'cr2', 'cr3', 'nef', 'nrw', 'arw', 'srf', 'sr2', 'orf', 'rw2', 'pef', 'dng',
  'raf', 'x3f', '3fr', 'fff', 'mef', 'mos', 'iiq', 'kdc', 'dcr', 'mrw', 'erf',
  'psd', 'psb', 'xcf', 'kra', 'clip', 'exr', 'hdr', 'tga', 'pcx', 'ppm', 'pgm', 'pbm',
  'pnm', 'wbmp', 'cur', 'ani',
])

// ── Audio ───────────────────────────────────────────────────────────────────
add('audio', [
  'mp3', 'wav', 'wave', 'flac', 'aac', 'm4a', 'm4b', 'm4p', 'ogg', 'oga', 'opus',
  'wma', 'aiff', 'aif', 'aifc', 'alac', 'caf', 'mid', 'midi', 'kar', 'rmi',
  'amr', 'awb', 'ac3', 'eac3', 'dts', 'dtshd', 'ape', 'wv', 'tta', 'tak',
  'ra', 'ram', 'au', 'snd', 'voc', 'gsm', 'spx', 'weba', 'mp2', 'mpa', 'mpc',
  'mod', 's3m', 'xm', 'it', 'nsf', 'sid', 'sap',
])

// ── Video ───────────────────────────────────────────────────────────────────
add('video', [
  'mp4', 'm4v', 'mov', 'qt', 'avi', 'mkv', 'webm', 'wmv', 'flv', 'f4v', 'f4p',
  'mpg', 'mpeg', 'mpe', 'mp2', 'm2v', 'm2ts', 'mts', 'vob', 'ogv', 'ogm',
  // note: do NOT include bare "ts" — that is TypeScript
  '3gp', '3g2', 'asf', 'rm', 'rmvb', 'divx', 'xvid', 'mxf', 'braw', 'r3d',
  'prores', 'dv', 'dvr-ms', 'wtv', 'yuv', 'y4m', 'nsv', 'amv', 'svi',
])

// ── Documents ───────────────────────────────────────────────────────────────
add('document', [
  'pdf', 'doc', 'docx', 'docm', 'dot', 'dotx', 'dotm', 'rtf', 'odt', 'ott', 'fodt',
  'txt', 'text', 'md', 'markdown', 'mdown', 'mkd', 'mdx', 'rst', 'adoc', 'asciidoc',
  'tex', 'latex', 'ltx', 'bib', 'pages', 'wpd', 'wps', 'abw', 'zabw', 'sxw',
  'log', 'nfo', 'readme', 'me', '1st', 'asc', 'csvnote', 'note', 'notes',
  'fountain', 'fdx', 'scriv', 'textile', 'org', 'wiki', 'creole',
])

// ── Spreadsheets ────────────────────────────────────────────────────────────
add('spreadsheet', [
  'xls', 'xlsx', 'xlsm', 'xlsb', 'xlt', 'xltx', 'xltm', 'ods', 'ots', 'fods',
  'csv', 'tsv', 'tab', 'numbers', 'dif', 'slk', 'sylk', 'dbf', 'prn', 'wq1',
  'qpw', 'wk1', 'wk3', 'wk4', '123', 'gsheet',
])

// ── Presentations ───────────────────────────────────────────────────────────
add('presentation', [
  'ppt', 'pptx', 'pptm', 'pot', 'potx', 'potm', 'pps', 'ppsx', 'ppsm',
  'odp', 'otp', 'fodp', 'key', 'keynote', 'gslides', 'sxi', 'sti',
])

// ── Archives / packages ─────────────────────────────────────────────────────
add('archive', [
  'zip', 'zipx', 'rar', '7z', 'tar', 'tgz', 'gz', 'gzip', 'bz2', 'bzip2', 'xz',
  'lz', 'lzma', 'lz4', 'zst', 'zstd', 'z', 'cab', 'arj', 'lzh', 'lha', 'ace',
  'arc', 'pak', 'pk3', 'pk4', 'war', 'ear', 'jar', 'aar', 'nupkg', 'crx',
  'xpi', 'vsix', 'whl', 'egg', 'gem', 'apk.backup', 'ipa.backup', 'sit', 'sitx',
  'sea', 'hqx', 'uu', 'uue', 'xxe', 'b64', 'cpgz', 'rp', 'shar', 'cbr', 'cbz',
  'cbt', 'cba', 'cb7',
])

// ── Apps / bundles ──────────────────────────────────────────────────────────
add('app', [
  'app', 'appbundle', 'appex', 'framework', 'plugin', 'kext', 'qlgenerator',
  'saver', 'wdgt', 'xpc', 'bundle', 'service',
])

// ── Disk images ─────────────────────────────────────────────────────────────
add('disk', [
  'dmg', 'iso', 'img', 'toast', 'cdr', 'dvd', 'bin', 'cue', 'mds', 'mdf', 'nrg',
  'vcd', 'vhd', 'vhdx', 'vmdk', 'vdi', 'qcow', 'qcow2', 'sparseimage', 'sparsebundle',
  'udf', 'wim', 'swm', 'esd',
])

// ── Installers / executables ────────────────────────────────────────────────
add('installer', [
  'exe', 'msi', 'msp', 'mst', 'msix', 'appx', 'appxbundle', 'msixbundle',
  'apk', 'aab', 'xapk', 'apks', 'ipa', 'deb', 'rpm', 'pkg', 'mpkg', 'flatpak',
  'snap', 'appimage', 'run', 'sh.install', 'bat', 'cmd', 'com', 'scr', 'pif',
  'gadget', 'cpl', 'msc', 'inf', 'inx', 'isu', 'job', 'jse', 'wsf', 'wsh',
  'ps1', 'psm1', 'psd1', 'vbs', 'vbe', 'ws', 'wsc',
])

// ── Programming / source ───────────────────────────────────────────────────
add('code', [
  // JS / TS
  'js', 'jsx', 'mjs', 'cjs', 'ts', 'tsx', 'mts', 'cts', 'vue', 'svelte', 'astro',
  'coffee', 'litcoffee', 'ls', 'livescript',
  // Python
  'py', 'pyw', 'pyi', 'pyx', 'pxd', 'pxi', 'ipynb', 'pyc', 'pyo', 'pyd',
  // Go / Rust / Zig / Nim
  'go', 'rs', 'rlib', 'zig', 'nim', 'nims', 'nimble',
  // JVM
  'java', 'jav', 'class', 'kt', 'kts', 'kotin', 'scala', 'sc', 'groovy', 'gradle',
  'clj', 'cljs', 'cljc', 'edn',
  // Apple / mobile
  'swift', 'm', 'mm', 'h', 'hpp', 'hxx', 'hh', 'c', 'cc', 'cpp', 'cxx', 'cp',
  'c++', 'inl', 'ixx', 'cppm',
  // .NET
  'cs', 'csx', 'fs', 'fsx', 'fsi', 'vb', 'vbscript', 'fsproj', 'csproj', 'vbproj',
  'sln', 'cshtml', 'razor', 'resx', 'xaml',
  // Ruby / PHP / Perl
  'rb', 'rbw', 'rake', 'gemspec', 'ru', 'erb', 'haml', 'slim',
  'php', 'php3', 'php4', 'php5', 'phtml', 'phps',
  'pl', 'pm', 't', 'pod', 'perl',
  // Functional / others
  'hs', 'lhs', 'elm', 'ex', 'exs', 'erl', 'hrl', 'beam',
  'ml', 'mli', 'mll', 'mly', 're', 'rei', 'res', 'resi',
  'dart', 'lua', 'luau', 'pd_lua', 'wlua',
  'r', 'rdata', 'rds', 'rda', 'jl', 'matlab', 'm',
  'f', 'for', 'f90', 'f95', 'f03', 'f08',
  'asm', 's', 'S', 'nasm', 'a51', 'inc',
  'v', 'vh', 'sv', 'svh', 'vhd', 'vhdl',
  'cob', 'cbl', 'cpy', 'ada', 'adb', 'ads',
  'pas', 'pp', 'dpr', 'dpk', 'lpr',
  'bas', 'bi', 'cls', 'frm',
  'prolog', 'pro', 'P',
  'lisp', 'lsp', 'cl', 'el', 'scm', 'ss', 'rkt',
  'tcl', 'tk', 'itk',
  'd', 'di',
  'cr', 'crystal',
  'hx', 'hxml',
  'purs', 'pure',
  'sol', 'cairo', 'move',
  'wgsl', 'glsl', 'vert', 'frag', 'comp', 'geom', 'tesc', 'tese',
  'hlsl', 'fx', 'fxh', 'cg', 'cginc',
  'shader', 'compute', 'metal',
  'gd', 'gdscript', 'gdshader',
  'unity', 'prefab', 'asset', 'meta', 'anim', 'controller',
  'uasset', 'umap',
  'wasm', 'wat',
  'nix', 'dhall', 'purs',
])

// ── Markup / stylesheets ────────────────────────────────────────────────────
add('markup', [
  'html', 'htm', 'xhtml', 'shtml', 'html5',
  'css', 'scss', 'sass', 'less', 'styl', 'stylus', 'pcss', 'postcss',
  'xml', 'xsl', 'xslt', 'xsd', 'dtd', 'rss', 'atom', 'svg',
  'jsp', 'jspx', 'asp', 'aspx', 'ascx', 'ashx', 'asmx',
  'ejs', 'hbs', 'handlebars', 'mustache', 'njk', 'nunjucks', 'pug', 'jade',
  'twig', 'liquid', 'jinja', 'j2', 'tmpl', 'tpl',
])

// svg already in image — markup add after may override. Prefer image for svg.
EXT_CATEGORY['svg'] = 'image'
EXT_CATEGORY['svgz'] = 'image'

// ── Data / interchange ─────────────────────────────────────────────────────
add('data', [
  'json', 'jsonc', 'json5', 'jsonl', 'ndjson', 'geojson', 'topojson',
  'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf', 'config', 'prefs', 'properties',
  'prop', 'props', 'plist', 'entitlements',
  'xml', 'xsd',
  'csv', 'tsv',
  'protobuf', 'proto', 'pb', 'pbtxt',
  'avro', 'avsc', 'parquet', 'orc', 'arrow', 'feather', 'ipc',
  'msgpack', 'cbor', 'bson', 'ubjson', 'ion',
  'graphql', 'gql', 'graphqls',
  'thrift', 'capnp', 'fbs', 'flatbuffers',
  'rdf', 'owl', 'ttl', 'n3', 'nt', 'nq', 'trig', 'jsonld',
  'kml', 'kmz', 'gpx', 'gml', 'shp', 'dbf', 'shx', 'prj',
  'fit', 'tcx',
])

// xml/csv may conflict — keep spreadsheet for csv/tsv, image already handles svg
EXT_CATEGORY['csv'] = 'spreadsheet'
EXT_CATEGORY['tsv'] = 'spreadsheet'
EXT_CATEGORY['xml'] = 'markup'

// ── Config / devops ─────────────────────────────────────────────────────────
add('config', [
  'env', 'env.local', 'env.development', 'env.production', 'env.test',
  'editorconfig', 'gitignore', 'gitattributes', 'gitmodules', 'gitkeep',
  'dockerignore', 'npmrc', 'yarnrc', 'pnpmrc', 'nvmrc', 'node-version',
  'babelrc', 'eslintrc', 'prettierrc', 'stylelintrc', 'browserslist',
  'dockerignore', 'htaccess', 'nginx', 'apache', 'htpasswd',
  'dockerfile', 'containerfile', 'compose', 'tf', 'tfvars', 'hcl', 'nomad',
  'jenkinsfile', 'procfile', 'buildpack', 'cfignore',
  'cmake', 'cmakelists', 'makefile', 'mak', 'mk', 'ninja', 'bazel', 'bzl',
  'buck', 'sbt', 'pom', 'gradle', 'kts',
  'sln', 'vcxproj', 'filters', 'user', 'suo',
  'pbxproj', 'xcworkspace', 'xcodeproj', 'xcconfig', 'entitlements',
  'plist', 'strings', 'xcprivacy',
  'service', 'socket', 'timer', 'target', 'mount', 'automount', 'path', 'slice',
  'desktop', 'service',
])

// ── Fonts ───────────────────────────────────────────────────────────────────
add('font', [
  'ttf', 'otf', 'woff', 'woff2', 'eot', 'fon', 'fnt', 'pfb', 'pfm', 'afm',
  'ttc', 'otc', 'dfont', 'suit', 'pfa', 'bdf', 'pcf', 'snf', 'psf',
])

// ── Ebooks ──────────────────────────────────────────────────────────────────
add('ebook', [
  'epub', 'mobi', 'azw', 'azw3', 'azw4', 'kfx', 'kf8', 'fb2', 'fb3', 'lit',
  'lrf', 'lrx', 'pdb', 'prc', 'tcr', 'cbz', 'cbr', 'ibooks', 'djvu', 'djv',
])

// ── Design ──────────────────────────────────────────────────────────────────
add('design', [
  'fig', 'figma', 'sketch', 'xd', 'afdesign', 'afphoto', 'afpub',
  'ai', 'ait', 'eps', 'epsf', 'epsi', 'indd', 'indt', 'idml', 'indb',
  'psd', 'psb', 'pdd', 'pdn', 'xcf', 'kra', 'ora', 'clip', 'sai', 'sai2',
  'cdr', 'cmx', 'svg', 'sketchpad',
  'blend', 'blend1', 'max', 'ma', 'mb', 'c4d', 'hip', 'hiplc',
  'aep', 'aet', 'prproj', 'ppj', 'fcp', 'fcpx', 'drp', 'veg',
  'fla', 'xfl', 'swf',
])

EXT_CATEGORY['svg'] = 'image'
EXT_CATEGORY['psd'] = 'design'
EXT_CATEGORY['ai'] = 'design'
EXT_CATEGORY['eps'] = 'design'

// ── 3D / CAD ────────────────────────────────────────────────────────────────
add('3d', [
  'obj', 'fbx', 'gltf', 'glb', 'stl', '3ds', 'dae', 'ply', 'x3d', 'wrl', 'vrml',
  'usd', 'usda', 'usdc', 'usdz', 'abc', '3mf', 'amf', 'step', 'stp', 'iges', 'igs',
  'dwg', 'dxf', 'dgn', 'sat', 'sab', 'ipt', 'iam', 'idw', 'prt', 'asm', 'sldprt',
  'sldasm', 'slddrw', 'catpart', 'catproduct', 'cgr', 'model', 'session',
  'skp', '3dm', 'rvt', 'rfa', 'rte', 'ifc', 'ifczip',
])

// ── Databases ───────────────────────────────────────────────────────────────
add('database', [
  'sql', 'mysql', 'pgsql', 'psql', 'plsql', 'db', 'db3', 'sqlite', 'sqlite3',
  'mdb', 'accdb', 'accde', 'accdt', 'accdr', 'fdb', 'gdb', 'nsf', 'fp7', 'fp5',
  'odb', 'dbf', 'frm', 'myd', 'myi', 'ibd', 'dump', 'backup', 'bak',
  'rdb', 'dump', 'bson',
])

// ── Certificates / keys ─────────────────────────────────────────────────────
add('cert', [
  'pem', 'crt', 'cer', 'der', 'p7b', 'p7c', 'p12', 'pfx', 'p8', 'pk8', 'pkcs12',
  'key', 'pub', 'priv', 'gpg', 'asc', 'sig', 'sign', 'csr', 'crl', 'sst', 'stl',
  'jks', 'keystore', 'truststore', 'bks', 'kdbx', 'kdb',
])

// ── Mail / calendar / contacts ──────────────────────────────────────────────
add('mail', [
  'eml', 'emlx', 'msg', 'oft', 'mbx', 'mbox', 'pst', 'ost', 'dbx', 'msf',
  'vcf', 'vcard', 'ldif',
])
add('calendar', [
  'ics', 'ical', 'icalendar', 'ifb', 'vcs', 'wks',
])

// ── Scripts / shells ────────────────────────────────────────────────────────
add('script', [
  'sh', 'bash', 'zsh', 'fish', 'ksh', 'csh', 'tcsh', 'command',
  'bat', 'cmd', 'ps1', 'psm1', 'psd1', 'ps1xml', 'psc1',
  'vbs', 'vbe', 'wsf', 'wsh', 'js', // js kept as code
  'rb', 'py', 'pl', 'php', // prefer code — reassert below
  'awk', 'sed', 'raku', 'p6', 'nsh', 'nsi', 'iss',
  'applescript', 'scpt', 'scptd', 'osascript',
])

// Prefer language categories for overlapping script extensions
for (const e of ['js', 'ts', 'tsx', 'jsx', 'py', 'rb', 'php', 'pl', 'go', 'rs']) {
  EXT_CATEGORY[e] = 'code'
}
EXT_CATEGORY['bat'] = 'installer'
EXT_CATEGORY['cmd'] = 'installer'
EXT_CATEGORY['ps1'] = 'script'
EXT_CATEGORY['sh'] = 'script'
EXT_CATEGORY['bash'] = 'script'
EXT_CATEGORY['zsh'] = 'script'
EXT_CATEGORY['mts'] = 'code'
EXT_CATEGORY['cts'] = 'code'

// ── Binary / unknown blobs ──────────────────────────────────────────────────
add('binary', [
  'bin', 'dat', 'data', 'blob', 'raw', 'rom', 'firmware', 'fw', 'hex', 'ihex',
  'srec', 'cof', 'elf', 'o', 'obj', 'a', 'lib', 'so', 'dylib', 'dll', 'sys',
  'drv', 'ko', 'class', 'jar', 'war', 'ear', 'node', 'napi',
  'pdb', 'ilk', 'exp', 'map', 'dSYM',
])

/** Infer category from vscode icon name for extensions not hand-listed above. */
function inferCategoryFromIcon(iconName: string): FileCategory {
  const n = iconName.toLowerCase()
  if (/(image|svg|webp|avif|photoshop|gimp|krita)/.test(n)) return 'image'
  if (/audio/.test(n)) return 'audio'
  if (/video/.test(n)) return 'video'
  if (/(zip|jar|nuget|vsix|archive)/.test(n)) return 'archive'
  if (/(excel|csv)/.test(n)) return 'spreadsheet'
  if (/(powerpoint|^file-type-key$)/.test(n)) return 'presentation'
  if (/(word|pdf|markdown|text|tex|asciidoc|log)/.test(n)) return 'document'
  if (/font/.test(n)) return 'font'
  if (/epub/.test(n)) return 'ebook'
  if (/(cert|gpg)/.test(n) || n === 'file-type-key') return 'cert'
  if (/(mail|outlook)/.test(n)) return 'mail'
  if (/(shell|powershell|bat|zsh|awk)/.test(n)) return 'script'
  if (/(db|sql|sqlite|access|mysql|pgsql|mongo)/.test(n)) return 'database'
  if (/(html|xml|css|sass|less|stylus|pug|ejs|handlebars)/.test(n)) return 'markup'
  if (/(json|yaml|toml|graphql|protobuf|avro|parquet|ini)/.test(n)) return 'data'
  if (/(docker|config|dotenv|terraform|cmake|makefile|git|editorconfig|npm|yarn|pnpm)/.test(n)) {
    return 'config'
  }
  if (/(sketch|figma|blender|ai$|indesign|flash|aseprite)/.test(n)) return 'design'
  if (/(gltf|fbx|obj)/.test(n)) return '3d'
  if (/(package|debian|binary)/.test(n)) return 'installer'
  if (
    /(typescript|javascript|python|rust|go|java|kotlin|swift|ruby|php|csharp|cpp|react|vue|svelte|dart|lua|haskell|elixir|erlang|scala|zig|wasm|solidity|fortran|perl|julia|ocaml|reason|nim|crystal|groovy|gradle|clojure|lisp|scheme|racket|matlab|r$|file-type-r)/.test(
      n,
    )
  ) {
    return 'code'
  }
  return 'other'
}

// Fill every generated logo extension so pills/labels work for ~1400+ types.
for (const [ext, icon] of Object.entries(generatedFileIconMap as Record<string, string>)) {
  if (!EXT_CATEGORY[ext]) {
    EXT_CATEGORY[ext] = inferCategoryFromIcon(icon)
  }
}

const FALLBACK: FileVisual = {
  icon: File,
  color: CATEGORY.other.color,
  bg: CATEGORY.other.bg,
  label: 'FILE',
  category: 'other',
}

/** Extract a usable path/name from a clipboard item's text fields. */
export function pathFromFileItem(item: {
  title?: string
  textPreview?: string
  textContent?: string
}): string {
  const raw = (item.textContent || item.textPreview || item.title || '').trim()
  const line = raw.split('\n').map((l) => l.trim()).find(Boolean) || ''
  return line
}

/** Last path segment extension, lowercased, without the dot. Multi-dot aware for a few cases. */
export function extensionFromPath(path: string): string {
  const base = path.split(/[/\\]/).pop() || path
  const name = base.split('?')[0].split('#')[0]
  if (!name || name.startsWith('.') && !name.slice(1).includes('.')) {
    // hidden file like `.gitignore` — treat whole name after dot as ext-ish
    if (name.startsWith('.') && name.length > 1) return name.slice(1).toLowerCase()
    return ''
  }
  const lower = name.toLowerCase()
  // Compound / special names
  const compounds = [
    'tar.gz', 'tar.bz2', 'tar.xz', 'tar.zst', 'tar.lz', 'tar.lz4',
    'user.js', 'd.ts', 'd.mts', 'd.cts',
    'test.ts', 'test.tsx', 'test.js', 'test.jsx',
    'spec.ts', 'spec.tsx', 'spec.js',
    'config.js', 'config.ts', 'config.cjs', 'config.mjs',
    'dockerfile', 'containerfile', 'makefile', 'cmakelists.txt',
    'jenkinsfile', 'procfile', 'gemfile', 'rakefile', 'brewfile',
    'vagrantfile', 'podfile', 'fastfile', 'appfile', 'matchfile',
  ]
  for (const c of compounds) {
    if (lower === c || lower.endsWith('.' + c) || lower.endsWith('/' + c)) {
      return c.includes('.') ? c.split('.').pop()! : c
    }
  }
  // .d.ts
  if (/\.d\.(ts|mts|cts)$/.test(lower)) return 'ts'
  const i = name.lastIndexOf('.')
  if (i <= 0 || i === name.length - 1) return ''
  return name.slice(i + 1).toLowerCase()
}

export function fileVisualFromExtension(ext: string): FileVisual {
  const e = ext.toLowerCase().replace(/^\./, '')
  if (!e) return FALLBACK
  const category = EXT_CATEGORY[e] || 'other'
  const style = CATEGORY[category]
  const icon = ICON_OVERRIDE[e] || style.icon
  return {
    icon,
    color: style.color,
    bg: style.bg,
    label: e.length <= 6 ? e.toUpperCase() : e.slice(0, 5).toUpperCase(),
    category,
  }
}

export function fileVisualFromPath(path: string): FileVisual {
  return fileVisualFromExtension(extensionFromPath(path))
}

export function fileVisualFromItem(item: {
  title?: string
  textPreview?: string
  textContent?: string
}): FileVisual {
  return fileVisualFromPath(pathFromFileItem(item))
}

/** Count of known extensions (for tests / debugging). */
export function knownExtensionCount(): number {
  return Object.keys(EXT_CATEGORY).length
}
