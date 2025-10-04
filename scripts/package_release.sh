#!/usr/bin/env bash
set -euo pipefail

# Build and package nebula_manager for multiple OS/Architecture targets.
# Usage: ./scripts/package_release.sh [version]

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

VERSION=${1:-$(git describe --tags --always 2>/dev/null || date +%Y%m%d)}
OUTPUT_DIR="$ROOT_DIR/build/packages"
BIN_NAME="nebula_manager"
FRONTEND_DIR="$ROOT_DIR/frontend"
DIST_DIR="$FRONTEND_DIR/dist"
TARGETS=(
  "linux/amd64"
  "linux/arm64"
  "linux/386"
  "windows/amd64"
  "darwin/amd64"
  "darwin/arm64"
)

mkdir -p "$OUTPUT_DIR"

# Build frontend bundle if missing
if [ ! -d "$DIST_DIR" ]; then
  echo "[frontend] dist/ missing, running npm install && npm run build ..."
  (cd "$FRONTEND_DIR" && npm install && npm run build)
fi

# Copy README and sample config once
STAGING_COMMON="$ROOT_DIR/build/_staging_common"
rm -rf "$STAGING_COMMON"
mkdir -p "$STAGING_COMMON"
cp README.md "$STAGING_COMMON/"
if [ -f "$ROOT_DIR/config.yaml.default" ]; then
  cp "$ROOT_DIR/config.yaml.default" "$STAGING_COMMON/"
fi
cp -R "$DIST_DIR" "$STAGING_COMMON/frontend"

for target in "${TARGETS[@]}"; do
  IFS=/ read -r GOOS GOARCH <<<"$target"
  echo "[build] targeting $GOOS/$GOARCH"

  STAGING_DIR="$ROOT_DIR/build/_staging_${GOOS}_${GOARCH}"
  rm -rf "$STAGING_DIR"
  mkdir -p "$STAGING_DIR"
  cp -R "$STAGING_COMMON"/* "$STAGING_DIR"

  EXT=""
  if [ "$GOOS" = "windows" ]; then
    EXT=".exe"
  fi

  OUTPUT_BIN="$STAGING_DIR/$BIN_NAME$EXT"
  env CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$OUTPUT_BIN" ./

  PACKAGE_NAME="nebula_manager_${VERSION}_${GOOS}_${GOARCH}"

  (cd "$STAGING_DIR" && \
    if [ "$GOOS" = "windows" ]; then
      zip -rq "$OUTPUT_DIR/${PACKAGE_NAME}.zip" .
    else
      tar -czf "$OUTPUT_DIR/${PACKAGE_NAME}.tar.gz" .
    fi
  )

  echo "  -> packaged $PACKAGE_NAME"
  rm -rf "$STAGING_DIR"
 done

rm -rf "$STAGING_COMMON"

echo "Packages available in $OUTPUT_DIR"
