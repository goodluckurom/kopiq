#!/usr/bin/env bash
# Interactive platform picker for production builds.
# Non-interactive: PLATFORM=darwin|windows|linux|macos-windows|all task build
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Task dynamic shell variables resolve before task-level env is applied.
# Export the bundled CLI path for those lookups and all nested task calls.
export PATH="$ROOT/bin:$PATH"

pick="${PLATFORM:-${1:-}}"

if [[ -z "$pick" ]]; then
  if [[ ! -t 0 ]]; then
    echo "No TTY and PLATFORM unset — building for host OS ($(go env GOOS))." >&2
    pick="$(go env GOOS)"
  else
    echo ""
    echo "  Kopiq production build"
    echo "  ──────────────────────"
    echo "  1) macOS"
    echo "  2) Windows"
    echo "  3) macOS + Windows"
    echo "  4) Linux"
    echo "  5) All (macOS + Windows + Linux)"
    echo "  6) This machine only ($(go env GOOS)/$(go env GOARCH))"
    echo "  7) Cancel"
    echo ""
    printf "  Choose [1-7]: "
    read -r choice
    case "$choice" in
      1) pick=darwin ;;
      2) pick=windows ;;
      3) pick=macos-windows ;;
      4) pick=linux ;;
      5) pick=all ;;
      6) pick="$(go env GOOS)" ;;
      7|"") echo "Cancelled."; exit 0 ;;
      *)
        echo "Invalid choice: $choice" >&2
        exit 1
        ;;
    esac
  fi
fi

# Aliases
case "$pick" in
  macos|mac|osx) pick=darwin ;;
  win) pick=windows ;;
  both) pick=macos-windows ;;
esac

# Use the project-pinned CLI instead of requiring a global Wails install.
if [[ ! -x "$ROOT/bin/wails3" ]]; then
  mkdir -p bin
  go build -o bin/wails3 github.com/wailsapp/wails/v3/cmd/wails3
fi

# Linux needs a container with its GTK/WebKit toolchain. Build it once for
# the requested architecture, rather than silently using a host-arch image.
if [[ "$pick" == linux || "$pick" == all ]]; then
  linux_arch="${LINUX_ARCH:-amd64}"
  needs_docker=true
  if [[ "$(go env GOOS)" == linux && "$linux_arch" == "$(go env GOARCH)" ]] && { command -v gcc >/dev/null 2>&1 || command -v clang >/dev/null 2>&1; }; then
    needs_docker=false
  fi
  if [[ "$needs_docker" == true ]] && ! docker info >/dev/null 2>&1; then
    echo "Docker is not running. Start Docker Desktop, then rerun task build." >&2
    exit 1
  fi
  image_arch="$(docker image inspect wails-cross --format '{{.Architecture}}' 2>/dev/null || true)"
  if [[ "$needs_docker" == true && "$image_arch" != "$linux_arch" ]]; then
    task setup:docker CROSS_ARCH="$linux_arch"
  fi
fi

run_one() {
  local os="$1"
  echo ""
  echo "==> Building for ${os}…"
  case "$os" in
    darwin)
      local arch="${MACOS_ARCH:-$(go env GOARCH)}"
      task darwin:build ARCH="$arch"
      cp bin/kopiq "bin/kopiq-darwin-$arch"
      ;;
    windows)
      local arch="${WINDOWS_ARCH:-amd64}"
      # Wails and Kopiq's Windows implementation support pure-Go builds.
      # macOS's required CGO_ENABLED=1 must not force Windows into Docker.
      task windows:build ARCH="$arch" CGO_ENABLED="${WINDOWS_CGO_ENABLED:-0}"
      if [[ "${WINDOWS_CGO_ENABLED:-0}" == 0 ]]; then
        cp bin/kopiq.exe "bin/kopiq-windows-$arch.exe"
      fi
      ;;
    linux)
      local arch="${LINUX_ARCH:-amd64}"
      task linux:build ARCH="$arch" OUTPUT="bin/kopiq-linux-$arch"
      ;;
    *)
      echo "Unknown platform: $os" >&2
      exit 1
      ;;
  esac
  echo "==> Done: ${os}"
}

case "$pick" in
  darwin|windows|linux)
    run_one "$pick"
    ;;
  macos-windows)
    run_one darwin
    run_one windows
    ;;
  all)
    run_one darwin
    run_one windows
    run_one linux
    ;;
  *)
    echo "Unknown PLATFORM='$pick' (use darwin|windows|linux|macos-windows|all)" >&2
    exit 1
    ;;
esac

echo ""
echo "Build finished."
ls -la bin/kopiq* bin/*.exe 2>/dev/null || true
