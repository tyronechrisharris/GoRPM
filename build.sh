#!/bin/bash

# Define binary name
APP_NAME="gorpm"
DIST_DIR="dist"

# Create dist directory
mkdir -p "$DIST_DIR"

# Common build flags
LDFLAGS="-s -w"

echo "Building for Apple Silicon (macOS/arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "$DIST_DIR/${APP_NAME}-macos-arm64" .

echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$DIST_DIR/${APP_NAME}-windows-x64.exe" .

echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$DIST_DIR/${APP_NAME}-linux-x64" .

echo "Building for Linux (386)..."
GOOS=linux GOARCH=386 go build -ldflags="$LDFLAGS" -o "$DIST_DIR/${APP_NAME}-linux-x86" .

echo "Build complete. Binaries are in the $DIST_DIR directory."
