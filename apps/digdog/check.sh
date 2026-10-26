#!/usr/bin/env bash
# Fast checks that need no GPU and no browser. Run from anywhere.
# Native Ebiten builds need graphics headers on Linux, so vet and build use the wasm target.
set -euo pipefail
cd "$(dirname "$0")"

echo "== gofmt =="
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "These files need formatting (run: gofmt -w <file>):"
  echo "$unformatted"
  exit 1
fi

echo "== go vet (wasm) =="
GOOS=js GOARCH=wasm go vet ./...

echo "== build (wasm) =="
GOOS=js GOARCH=wasm go build -o /tmp/digdog-check.wasm .

echo "== tests =="
if compgen -G "internal/*/*_test.go" > /dev/null; then
  go test ./internal/...
else
  echo "(no tests yet)"
fi

echo "OK"
