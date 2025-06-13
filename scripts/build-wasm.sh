#!/bin/bash
set -euo pipefail

# ANSI colors
RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

info()    { echo "$1"; }
success() { echo -e "${GREEN}$1${RESET}"; }
error()   { echo -e "${RED}$1${RESET}"; exit 1; }

info "Building Bloblang WASM Playground..."

# Config
ENTRY="./cmd/wasm/main.go"
OUT_DIR="website/static/wasm"
WASM_BIN="${OUT_DIR}/bloblang-playground.wasm"
WASM_EXEC="${OUT_DIR}/wasm_exec.js"
WASM_EXEC_SRC="$(go env GOROOT)/misc/wasm/wasm_exec.js"

# Create output directory
mkdir -p "$OUT_DIR"

info "Compiling Go to WASM..."
GOOS=js GOARCH=wasm go build -o "$WASM_BIN" "$ENTRY"
[ -f "$WASM_BIN" ] || error "WASM binary was not created: $WASM_BIN"

info "Copying wasm_exec.js..."
cp "$WASM_EXEC_SRC" "$WASM_EXEC"
[ -f "$WASM_EXEC" ] || error "Failed to copy wasm_exec.js to $WASM_EXEC"

echo ""
success "WASM build complete."
echo ""
info "Files created:"
for f in "$WASM_BIN" "$WASM_EXEC"; do
    size=$(du -h "$f" | cut -f1)
    echo " - $f ($size)"
done

echo ""
info "To test the WASM playground locally:"
echo "  Serve static files using Python's simple HTTP server:"
echo "    cd \"$OUT_DIR\" && python3 -m http.server 8000"
echo ""
echo "  Open the standalone playground in your browser:"
echo "    http://localhost:8000/bloblang-playground.html"
echo ""
echo "  Alternatively, run the full docs site:"
echo "    cd website && yarn run start"
echo ""
echo "  Then open the integrated playground here:"
echo "    http://localhost:3000/bento/docs/guides/bloblang/playground"
