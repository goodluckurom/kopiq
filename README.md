# Kopiq

Local-first clipboard manager for developers.

```text
kopiq/
├── desktop/          # Wails v3 + Go + React desktop app
├── docs/             # Product spec & feature trackers
├── design/           # UI mockups & design notes
├── checklist.md      # Engineering handoff checklist
├── branding.png      # Brand identity sheet
└── icon.png          # App mark
```

## Authoritative sources

| Source | Role |
| --- | --- |
| [docs/](docs/README.md) | Product spec, architecture, feature trackers |
| [checklist.md](checklist.md) | Engineering handoff checklist |
| [branding.png](branding.png) | Brand (Lime `#A3FF12`, Charcoal `#0F1110`) |
| [icon.png](icon.png) | App mark |
| [design/](design/) | UI mockups |

## Desktop app

Everything runnable lives in **`desktop/`**.

```bash
cd desktop
task setup   # once
task dev     # real window + HMR
```

| Task | Purpose |
| --- | --- |
| `task dev` | Desktop app with Vite HMR |
| `task ui` | Browser-only UI preview |
| `task bindings` | Regenerate Go → TS bindings |
| `task build` / `task run` | Production-style binary |
| `task reset:data` | Clear `./.data` clipboard DB |

Env (auto-loaded from `desktop/.env` via Taskfile): `KOPIQ_DATA_DIR`, `GOMODCACHE`, `GOPATH`, `CGO_ENABLED`, `WAILS_VITE_PORT`.

If Go module unzip fails on `.vscode` paths:

```bash
cd desktop && python3 scripts/gomod-extract.py
```

## Stack

- **Desktop:** Wails v3 · Go · React · TypeScript · Vite · Tailwind
- **Storage:** SQLite + local blobs
- **Not using:** Electron, mandatory cloud, mandatory AI
