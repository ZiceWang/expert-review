#!/bin/bash
set -e

ORIG_DIR="$(pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="$SCRIPT_DIR/build"
MCP_SERVER="$SCRIPT_DIR/mcp-server"
FRONTEND="$SCRIPT_DIR/frontend"

echo "================================"
echo "Expert Review MCP Server Build"
echo "================================"
echo ""

# Clean and create build dir
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/public"

echo "[1/4] Go dependencies..."
cd "$MCP_SERVER"
go mod tidy

echo ""
echo "[2/4] Building Go server..."
cd "$MCP_SERVER"
go build -o "$BUILD_DIR/expert-review" .

echo ""
echo "[3/4] Frontend dependencies..."
cd "$FRONTEND"
npm install

echo ""
echo "[4/4] Building Vue frontend..."
cd "$FRONTEND"
npm run build

echo ""
echo "[5/5] Copying frontend to build/public..."
cp -r "$FRONTEND/dist/." "$BUILD_DIR/public/"

echo ""
echo "================================"
echo "Build complete!"
echo "================================"
echo ""
echo "Output: $BUILD_DIR/"
echo "  - expert-review        (Go server)"
echo "  - public/            (Vue frontend)"
echo ""
echo "To run:"
echo "  cd $BUILD_DIR"
echo "  ./expert-review"
echo ""
echo "Frontend: http://localhost:3100"
echo "================================"

cd "$ORIG_DIR"
