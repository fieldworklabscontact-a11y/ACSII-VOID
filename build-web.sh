set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="$ROOT_DIR/dist"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"
cp "$ROOT_DIR/web/index.html" "$DIST_DIR/"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$DIST_DIR/"
cd "$ROOT_DIR"
GOOS=js GOARCH=wasm go build -o "$DIST_DIR/game.wasm"
