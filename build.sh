#!/bin/bash

# 设置变量
BINARY_NAME="iptv-aggregator"
VERSION="1.0.0"
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
LDFLAGS="-X 'main.Version=$VERSION' -X 'main.BuildTime=$BUILD_TIME' -s -w"

echo "========================================"
echo "  IPTV M3U Aggregator Build Script (Linux)"
echo "========================================"

# 创建目录
mkdir -p build/windows
mkdir -p build/debian

echo "[1/2] Building for Windows x64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o build/windows/$BINARY_NAME.exe .

# 拷贝资源
echo "  Copying resources for Windows..."
cp -r templates build/windows/
cp -r static build/windows/
mkdir -p build/windows/data
[ -f config.json ] && cp config.json build/windows/

echo "[2/2] Building for Debian (Linux x64)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o build/debian/$BINARY_NAME .

# 拷贝资源
echo "  Copying resources for Debian..."
cp -r templates build/debian/
cp -r static build/debian/
mkdir -p build/debian/data
[ -f config.json ] && cp config.json build/debian/

echo ""
echo "========================================"
echo "  Build Successful!"
echo "  Windows: build/windows/"
echo "  Debian:  build/debian/"
echo "========================================"
