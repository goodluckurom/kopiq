#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/frontend/dist"
mkdir -p "$DIST"
if [[ ! -f "$DIST/index.html" ]]; then
  cat > "$DIST/index.html" <<'EOF'
<!doctype html>
<html>
  <head><meta charset="utf-8"><title>Kopiq</title></head>
  <body style="background:#0F1110;color:#E9ECEB;font-family:system-ui;padding:2rem">Kopiq</body>
</html>
EOF
  echo "Wrote $DIST/index.html stub"
fi
