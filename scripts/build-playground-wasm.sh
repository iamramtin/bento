#!/usr/bin/env bash
set -euo pipefail

# --- Config ---
readonly ENTRY="./cmd/wasm/main.go"
readonly OUT_DIR="website/static/playground"
readonly WASM_BIN="${OUT_DIR}/playground.wasm"
readonly WASM_EXEC="${OUT_DIR}/wasm_exec.js"
readonly WASM_EXEC_SRC="$(go env GOROOT)/misc/wasm/wasm_exec.js"

# --- Flags ---
VERBOSE="${VERBOSE:-false}"
CLEAN="${CLEAN:-false}"

# --- Colours ---
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly RESET='\033[0m'

log()     { echo -e "$1"; }
info()    { echo -e "${BLUE}$1${RESET}"; }
success() { echo -e "${GREEN}$1${RESET}"; }
error()   { echo -e "${RED}$1${RESET}"; exit 1; }

usage() {
    echo "Usage: $0 [--clean] [--verbose] [-h|--help]"
    echo "  --clean     Remove previous build output before building"
    echo "  --verbose   Enable verbose logging"
    echo "  -h, --help  Show this help message"
    exit 0
}

# --- Parse Args ---
while getopts ":chv" opt; do
    case "$opt" in
        c) CLEAN="true" ;;
        v) VERBOSE="true" ;;
        h) usage ;;
        *) error "Invalid option";;
    esac
done

# --- Checks ---
command -v go >/dev/null 2>&1 || error "Go is not installed or not in PATH."
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
GO_VERSION_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
GO_VERSION_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)
if [[ "$GO_VERSION_MAJOR" -lt 1 ]] || { [[ "$GO_VERSION_MAJOR" -eq 1 ]] && [[ "$GO_VERSION_MINOR" -lt 11 ]]; }; then
    error "Go 1.11 or newer is required for WASM support. Found: $GO_VERSION"
fi
[[ -f "$ENTRY" ]] || error "Entry file not found: $ENTRY"
[[ -f "$WASM_EXEC_SRC" ]] || error "wasm_exec.js not found at: $WASM_EXEC_SRC"

# --- Clean ---
if [[ "$CLEAN" == "true" ]]; then
    info "Cleaning previous build output in \"$OUT_DIR\""
    rm -rf "$OUT_DIR"
fi

# --- File size helper ---
get_file_size() {
    local file="$1"
    if [[ -f "$file" ]]; then
        if command -v stat >/dev/null 2>&1; then
            if stat --version >/dev/null 2>&1; then
                stat --format="%s bytes" "$file"
            else
                stat -f "%z bytes" "$file"
            fi
        else
            du -h "$file" | cut -f1
        fi
    else
        echo "N/A"
    fi
}


# --- Main Build ---
info "Building WASM playground...\n"

mkdir -p "$OUT_DIR" && log "Created output directory: \"$OUT_DIR\""

log "Compiling Go to WASM with: GOOS=js GOARCH=wasm go build -o \"$WASM_BIN\" \"$ENTRY\""
GOOS=js GOARCH=wasm go build -o "$WASM_BIN" "$ENTRY"
[[ -f "$WASM_BIN" ]] || error "WASM binary was not created: $WASM_BIN"
log "WASM binary created: \"$WASM_BIN\" ($(get_file_size "$WASM_BIN"))"

log "Copying wasm_exec.js from: \"$WASM_EXEC_SRC\""
cp "$WASM_EXEC_SRC" "$WASM_EXEC"
[[ -f "$WASM_EXEC" ]] || error "Failed to copy wasm_exec.js to $WASM_EXEC"
log "JavaScript support file copied: \"$WASM_EXEC\" ($(get_file_size "$WASM_EXEC"))"

success "\nWASM build complete"

# --- Verbose Output ---
if [[ "$VERBOSE" == "true" ]]; then
    echo ""
    info "Files created:"
    echo "   $WASM_BIN ($(get_file_size "$WASM_BIN"))"
    echo "   $WASM_EXEC ($(get_file_size "$WASM_EXEC"))"

    # Check for JS files
    if [[ -d "${OUT_DIR}/js" ]]; then
        echo ""
        info "JavaScript files:"
        for js_file in "${OUT_DIR}/js"/*.js; do
            [[ -f "$js_file" ]] && echo "   $js_file ($(get_file_size "$js_file"))"
        done
    fi

    # Check for CSS files
    if [[ -d "${OUT_DIR}/assets/css" ]]; then
        echo ""
        info "CSS files:"
        for css_file in "${OUT_DIR}/assets/css"/*.css; do
            [[ -f "$css_file" ]] && echo "   $css_file ($(get_file_size "$css_file"))"
        done
    fi

    echo ""
    info "To use locally:"
    echo "  Option 1: Serve the static playground directly"
    echo "    (You can use Python, GoExec, or any local HTTP server)"
    echo "      cd \"$OUT_DIR\" && python3 -m http.server 8000"
    echo ""
    echo "    Then open in your browser:"
    echo "      http://localhost:8000/playground.html"
    echo ""
    echo "  Option 2: Run the full documentation site with embedded playground"
    echo "      cd website && yarn start"
    echo ""
    echo "    Then open in your browser:"
    echo "      http://localhost:3000/bento/docs/guides/bloblang/playground"
fi

# For more info, see: https://go.dev/wiki/WebAssembly