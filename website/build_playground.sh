#!/bin/bash

# Optimized WASM build script for GitHub Actions deployment

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLAYGROUND_DIR="$ROOT_DIR/../internal/cli/blobl/playground"
STATIC_DIR="$ROOT_DIR/static/playground"
TEMP_WASM="/tmp/playground.wasm"
MAIN_GO="$ROOT_DIR/../cmd/playground/main.go"

# Build WASM with optimization flags
echo "Building Playground (WASM binary)..."
GOOS=js GOARCH=wasm go build \
    -ldflags="-s -w -extldflags=-static" \
    -tags="netgo,osusergo,static_build" \
    -trimpath \
    -gcflags="-l=4" \
    -o "$TEMP_WASM" "$MAIN_GO"

echo "Binary size:$(du -h "$TEMP_WASM" | cut -f1)"

# Download wasm_exec.js
echo "Downloading WASM runtime..."
WASM_EXEC_URL="https://raw.githubusercontent.com/golang/go/go$(go version | cut -d' ' -f3 | sed 's/go//')/misc/wasm/wasm_exec.js"
TEMP_WASM_EXEC="/tmp/wasm_exec.js"
curl -sf "$WASM_EXEC_URL" -o "$TEMP_WASM_EXEC" || {
    echo "Failed to download wasm_exec.js"
    exit 1
}

# Copy playground files and add WASM artifacts
echo "Preparing static files..."
rm -rf "$STATIC_DIR"
cp -r "$PLAYGROUND_DIR" "$STATIC_DIR"

# Copy WASM runtime
cp "$TEMP_WASM_EXEC" "$STATIC_DIR/js/wasm_exec.js"

# Copy WASM file
cp "$TEMP_WASM" "$STATIC_DIR/playground.wasm"
echo "WASM file: $(du -h "$STATIC_DIR/playground.wasm" | cut -f1)"

# Apply WASM mode configuration
echo "Configuring for WASM mode..."
sed -i.bak \
    's/window\.BLOBLANG_SYNTAX = {{\.BloblangSyntax}};/window.BLOBLANG_SYNTAX = undefined; \/\/ Will be loaded via WASM getBloblangSyntax()/g' \
    "$STATIC_DIR/index.html" && rm "$STATIC_DIR/index.html.bak"

# Clean up temp files
rm -f "$TEMP_WASM" "$TEMP_WASM_EXEC"

echo "✔ Playground ready for deployment!"
echo "⨉ Final: playground.wasm ($(du -h "$STATIC_DIR/playground.wasm" | cut -f1))"