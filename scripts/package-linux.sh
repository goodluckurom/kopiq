#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
format="${1:?Package format is required}"
binary="${2:?Linux binary is required}"
arch="${3:?Architecture is required}"
config="$(mktemp "${TMPDIR:-/tmp}/kopiq-package.XXXXXX")"
trap 'rm -f "$config"' EXIT
# NFPM expands arch variables, but not content source paths. Render only
# these two known values, and reject a macOS/Windows or wrong-arch payload.
python3 - "$ROOT/build/linux/nfpm/nfpm.yaml" "$config" "$binary" "$arch" <<'PYCODE'
import json, struct, sys
from pathlib import Path
source, target, binary, arch = sys.argv[1:]
header = Path(binary).read_bytes()[:20]
expected = {"amd64": 62, "arm64": 183}.get(arch)
if header[:4] != b"\x7fELF" or expected is None or struct.unpack("<H", header[18:20])[0] != expected:
    raise SystemExit("Package input must be the matching Linux ELF binary.")
text = Path(source).read_text().replace('"${KOPIQ_LINUX_BINARY}"', json.dumps(binary)).replace('${GOARCH}', json.dumps(arch))
Path(target).write_text(text)
PYCODE
"$ROOT/bin/wails3" tool package -name kopiq -format "$format" -config "$config" -out "$ROOT/bin"
