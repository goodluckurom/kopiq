#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if command -v makensis >/dev/null 2>&1; then
  exec makensis "$@"
fi
if ! docker info >/dev/null 2>&1; then
  echo "Install makensis or start Docker Desktop to package the Windows installer." >&2
  exit 1
fi
if ! docker image inspect kopiq-nsis >/dev/null 2>&1; then
  docker build -t kopiq-nsis -f "$ROOT/build/docker/Dockerfile.nsis" "$ROOT/build/docker"
fi
# Preserve host absolute paths in the NSIS arguments and relative include paths.
exec docker run --rm -v "$ROOT:$ROOT" -w "$PWD" kopiq-nsis "$@"
