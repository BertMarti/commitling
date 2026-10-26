#!/usr/bin/env bash
# Builds the precompiled binaries of a release, reproducibly: the same source
# and the same Go (the go line of go.mod) give the same bytes on any machine,
# so their SHA-256 can be committed BEFORE the release exists.
#
#   scripts/build-release.sh <dir>            build into <dir> and write <dir>/checksums.txt
#   scripts/build-release.sh <dir> --write    ...and copy it to release/checksums.txt (commit it, then publish the release)
#   scripts/build-release.sh <dir> --check    ...and fail unless it matches release/checksums.txt
#
# The Action only downloads a binary whose SHA-256 is listed in the
# release/checksums.txt of its own tree; without that file it compiles with
# `go run`, as it always did. The release workflow (.github/workflows/release.yml)
# runs this script with --check and uploads the binaries to the release.
set -euo pipefail
# The environment must not change the bytes: GOFLAGS (e.g. -tags, which the
# build info records) and GOEXPERIMENT are reset; -p only sets parallelism
# (2 for the shared Raspberry Pi) and does not alter the output.
export GOFLAGS=-p=2 GOEXPERIMENT= GOFIPS140=off

out="${1:?uso: scripts/build-release.sh <carpeta> [--write|--check]}"
mode="${2:-}"
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

version="$(sed -n 's/^var version = "\(.*\)"$/\1/p' cmd/commitling/main.go)"
[ -n "$version" ] || { echo "no encuentro la versión en cmd/commitling/main.go" >&2; exit 1; }
want_go="go$(awk '$1 == "go" {print $2}' go.mod)"
have_go="$(go env GOVERSION)"
if [ "$have_go" != "$want_go" ]; then
  echo "hace falta $want_go (la línea go de go.mod) y está $have_go: con otro Go los binarios no coinciden" >&2
  exit 1
fi

mkdir -p "$out"
rm -f "$out"/commitling_* "$out/checksums.txt"
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  goos="${target%/*}"
  goarch="${target#*/}"
  name="commitling_${version}_${goos}_${goarch}"
  [ "$goos" = windows ] && name="$name.exe"
  # Everything that could differ between machines is fixed: no cgo, no VCS
  # stamp, no paths (-trimpath), no build ID, baseline CPU levels.
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOAMD64=v1 GOARM64=v8.0 \
    go build -buildvcs=false -trimpath -ldflags="-s -w -buildid=" -o "$out/$name" ./cmd/commitling
done

if command -v sha256sum >/dev/null 2>&1; then sum() { sha256sum "$@"; }; else sum() { shasum -a 256 "$@"; }; fi
(cd "$out" && sum commitling_* | sed 's/ \*/  /') > "$out/checksums.txt"
cat "$out/checksums.txt"

case "$mode" in
  --write)
    mkdir -p release
    cp "$out/checksums.txt" release/checksums.txt
    echo "release/checksums.txt escrito: súbelo en un commit y publica la release v$version desde ese commit."
    ;;
  --check)
    if ! diff -u release/checksums.txt "$out/checksums.txt"; then
      echo "los binarios no coinciden con release/checksums.txt (¿cambió el código o la versión de Go sin regenerarlo?)" >&2
      exit 1
    fi
    echo "Los binarios coinciden con release/checksums.txt."
    ;;
  "") ;;
  *) echo "opción desconocida: $mode" >&2; exit 2 ;;
esac
