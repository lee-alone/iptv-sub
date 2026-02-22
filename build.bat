@echo off
setlocal enabledelayedexpansion

:: 设置变量
set BINARY_NAME=iptv-aggregator
set VERSION=1.0.0
set BUILD_TIME=%date% %time%
set LDFLAGS=-ldflags "-X main.Version=%VERSION% -X main.BuildTime=now -s -w"

echo ========================================
echo   IPTV M3U Aggregator Build Script (Win)
echo   [Single Binary Mode with go:embed]
echo ========================================

:: 创建 build 目录
if not exist "build" mkdir build
if not exist "build\windows" mkdir build\windows
if not exist "build\debian" mkdir build\debian

echo [1/2] Building for Windows x64...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build %LDFLAGS% -o build/windows/%BINARY_NAME%.exe .
if %errorlevel% neq 0 (
    echo Error building for Windows
    pause
    exit /b %errorlevel%
)

:: 单二进制模式下不再拷贝 templates 和 static
echo   Copying runtime config...
if exist "config.json" copy /y "config.json" "build\windows\" >nul

echo [2/2] Building for Debian (Linux x64)...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build %LDFLAGS% -o build/debian/%BINARY_NAME% .
if %errorlevel% neq 0 (
    echo Error building for Debian
    pause
    exit /b %errorlevel%
)

echo   Copying runtime config...
if exist "config.json" copy /y "config.json" "build\debian\" >nul

echo.
echo ========================================
echo   Build Successful!
echo   Outputs are in build/windows/ and build/debian/
echo   (Static assets are embedded in the binary)
echo ========================================
pause
