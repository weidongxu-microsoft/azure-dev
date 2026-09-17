#!/bin/bash
set -e

EXTENSION_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$EXTENSION_DIR"

EXTENSION_ID_SAFE="${EXTENSION_ID//./-}"
OUTPUT_DIR="${OUTPUT_DIR:-$EXTENSION_DIR/bin}"
mkdir -p "$OUTPUT_DIR"

if [ -n "$EXTENSION_PLATFORM" ]; then
    PLATFORMS=("$EXTENSION_PLATFORM")
else
    PLATFORMS=(
        "windows/amd64"
        "windows/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "linux/amd64"
        "linux/arm64"
    )
fi

for PLATFORM in "${PLATFORMS[@]}"; do
    OS="${PLATFORM%/*}"
    ARCH="${PLATFORM#*/}"
    OUTPUT_NAME="$OUTPUT_DIR/$EXTENSION_ID_SAFE-$OS-$ARCH"
    if [ "$OS" = "windows" ]; then
        OUTPUT_NAME="$OUTPUT_NAME.exe"
    fi

    GOOS="$OS" GOARCH="$ARCH" go build -o "$OUTPUT_NAME"
done
