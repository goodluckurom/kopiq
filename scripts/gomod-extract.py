#!/usr/bin/env python3
"""Extract Go module zips into GOMODCACHE while skipping hidden path components.

Cursor's sandbox blocks creating directories like .vscode / .github inside
the module cache. Run this after `go get` / `go mod tidy` fails on unzip.
"""
from __future__ import annotations

import os
import zipfile
from pathlib import Path


def blocked(path: str) -> bool:
    return any(part.startswith(".") and part not in (".", "..") for part in Path(path).parts)


def main() -> None:
    cache = Path(os.environ.get("GOMODCACHE", Path.cwd() / ".gomodcache")).resolve()
    download = cache / "cache" / "download"
    if not download.exists():
        print(f"no download cache at {download}")
        return

    total_e = total_s = 0
    for zpath in download.rglob("*.zip"):
        with zipfile.ZipFile(zpath) as zf:
            for name in zf.namelist():
                if blocked(name):
                    total_s += 1
                    continue
                target = cache / name
                if name.endswith("/"):
                    target.mkdir(parents=True, exist_ok=True)
                    continue
                try:
                    target.parent.mkdir(parents=True, exist_ok=True)
                    with zf.open(name) as src, open(target, "wb") as dst:
                        dst.write(src.read())
                    total_e += 1
                except PermissionError:
                    total_s += 1
    print(f"extracted={total_e} skipped={total_s} cache={cache}")


if __name__ == "__main__":
    main()
