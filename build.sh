#!/usr/bin/env bash
# Zero-dependency build script — use this if `make` isn't installed.
# Usage: ./build.sh [run|test|clean]
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

BINARY=triage

case "${1:-build}" in
  build)
    go build -o "$BINARY" ./cmd/triage
    ;;
  run)
    go build -o "$BINARY" ./cmd/triage
    ./"$BINARY"
    ;;
  test)
    go test ./...
    ;;
  clean)
    rm -f "$BINARY"
    ;;
  *)
    echo "usage: $0 [build|run|test|clean]" >&2
    exit 1
    ;;
esac
