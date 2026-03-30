#!/bin/bash
set -e

ORIG_DIR="$(pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASES_DIR="$SCRIPT_DIR/releases"
MCP_SERVER="$SCRIPT_DIR/mcp-server"
FRONTEND="$SCRIPT_DIR/frontend"
VERSION="0.0.1"

# Targets: OS/ARCH
TARGETS=(
  "darwin/amd64"
  "darwin/arm64"
  "freebsd/amd64"
  "linux/386"
  "linux/amd64"
  "linux/arm"
  "linux/arm64"
  "linux/riscv64"
  "openbsd/amd64"
  "windows/386"
  "windows/amd64"
  "windows/arm64"
)

echo "================================"
echo "Cross-compile Expert Review MCP"
echo "================================"
echo ""

# Clean releases dir
rm -rf "$RELEASES_DIR"
mkdir -p "$RELEASES_DIR"

# Build frontend once
echo "[1/3] Building frontend..."
cd "$FRONTEND"
npm install
npm run build

echo ""
echo "[2/3] Cross-compiling Go server..."

for TARGET in "${TARGETS[@]}"; do
  OS="${TARGET%/*}"
  ARCH="${TARGET#*/}"
  BASENAME="expert_review-${VERSION}-${OS}_${ARCH}"
  OUTPUT_NAME="expert-review"
  [ "$OS" = "windows" ] && OUTPUT_NAME="expert-review.exe"

  echo "  Building $BASENAME..."

  # Build
  cd "$MCP_SERVER"
  GOOS="$OS" GOARCH="$ARCH" go build -o "$RELEASES_DIR/$BASENAME/$OUTPUT_NAME" .

  # Copy public (frontend)
  mkdir -p "$RELEASES_DIR/$BASENAME/public"
  cp -r "$FRONTEND/dist/." "$RELEASES_DIR/$BASENAME/public/"

  # Create zip
  cd "$RELEASES_DIR"
  rm -f "${BASENAME}.zip"
  if [ "$OS" = "windows" ]; then
    powershell -Command "Compress-Archive -Path '${BASENAME}/*' -DestinationPath '${BASENAME}.zip' -Force"
  else
    cd "$BASENAME"
    zip -r "../${BASENAME}.zip" .
    cd ..
  fi

  rm -rf "$BASENAME"
  echo "  -> ${BASENAME}.zip"
done

echo ""
echo "[3/3] Summary"
echo "================================"
ls -lh "$RELEASES_DIR"/*.zip 2>/dev/null

cd "$ORIG_DIR"
echo ""
echo "Done: $RELEASES_DIR"
