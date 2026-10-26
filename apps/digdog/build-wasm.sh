#!/usr/bin/env bash
# Builds the game for the browser. wasm_exec.js must come from the same Go version.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p web
GOOS=js GOARCH=wasm go build -o web/digdog.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
echo "built web/digdog.wasm"
