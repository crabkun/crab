#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-2.0.0}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
MODULE="github.com/crabkun/crab"
OUT="${ROOT}/dist"

LDFLAGS="-s -w \
  -X ${MODULE}/internal/version.Version=${VERSION} \
  -X ${MODULE}/internal/version.Commit=${COMMIT} \
  -X ${MODULE}/internal/version.Date=${DATE}"

rm -rf "$OUT"
mkdir -p "$OUT"

pack() {
  local goos="$1" goarch="$2" extra="${3:-}"
  local label="${goos}-${goarch}"
  local ext=""
  if [[ -n "$extra" ]]; then
    # extra is GOARM, e.g. 7
    label="${goos}-armv${extra}"
  fi
  if [[ "$goos" == windows ]]; then
    ext=".exe"
  fi

  local name="crab-${VERSION}-${label}"
  local work="${OUT}/.work/${name}"
  mkdir -p "$work"

  echo "building ${name}"
  env CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOARM="$extra" \
    go build -trimpath -ldflags "$LDFLAGS" -o "${work}/crab${ext}" ./cmd/crab

  cp LICENSE README.md "$work/"
  cp -R config-examples "$work/"

  if [[ "$goos" == windows ]]; then
    (cd "${OUT}/.work" && zip -qry "${OUT}/${name}.zip" "$name")
  else
    tar -C "${OUT}/.work" -czf "${OUT}/${name}.tar.gz" "$name"
  fi
}

pack linux amd64
pack linux arm64
pack linux arm 7
pack darwin amd64
pack darwin arm64
pack windows amd64
pack windows arm64

rm -rf "${OUT}/.work"

(
  cd "$OUT"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 crab-* > SHA256SUMS
  else
    sha256sum crab-* > SHA256SUMS
  fi
)

echo "artifacts in ${OUT}:"
ls -lh "$OUT"
