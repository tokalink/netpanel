#!/bin/bash

# Define output directory
OUTPUT_DIR="build"
mkdir -p $OUTPUT_DIR

echo "🚀 Starting cross-platform build..."

# Windows (amd64)
echo "📦 Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $OUTPUT_DIR/vps-panel-windows-amd64.exe ./cmd/server

# Linux (amd64)
echo "🐧 Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $OUTPUT_DIR/vps-panel-linux-amd64 ./cmd/server

# Linux (arm64)
echo "🐧 Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $OUTPUT_DIR/vps-panel-linux-arm64 ./cmd/server

# macOS (Intel)
echo "🍎 Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $OUTPUT_DIR/vps-panel-darwin-amd64 ./cmd/server

# macOS (Apple Silicon/M1)
echo "🍎 Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $OUTPUT_DIR/vps-panel-darwin-arm64 ./cmd/server

echo "✅ Build complete! Binaries are in the '$OUTPUT_DIR' directory."
ls -lh $OUTPUT_DIR