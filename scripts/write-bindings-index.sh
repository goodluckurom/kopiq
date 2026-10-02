#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/frontend/bindings/kopiq/index.ts"
mkdir -p "$(dirname "$OUT")"
cat > "$OUT" <<'EOF'
// Re-export bound services for the frontend (kept after `wails3 generate bindings -clean`).
export * as HistoryService from './service/historyservice'
export * as SearchService from './service/searchservice'
export * as CaptureService from './service/captureservice'
export * as SettingsService from './service/settingsservice'
export type * from './internal/domain/models'
EOF
echo "Wrote $OUT"
