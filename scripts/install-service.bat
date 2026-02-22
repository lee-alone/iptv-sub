@echo off
REM IPTV Aggregator Service Installation Script (Windows)
REM Note: Service management is only supported on Linux
REM This script is provided for reference on Windows systems

setlocal enabledelayedexpansion

echo.
echo ========================================
echo IPTV Aggregator Service Installation
echo ========================================
echo.
echo NOTE: Service management is only supported on Linux systems.
echo.
echo On Linux, you can install the service using:
echo   sudo ./iptv-aggregator -s install
echo.
echo For detailed instructions, see:
echo   docs/SERVICE_MANAGEMENT.md
echo.
echo Available service commands (Linux only):
echo   -s install      Install as system service
echo   -s uninstall    Uninstall system service
echo   -s start        Start service
echo   -s stop         Stop service
echo   -s restart      Restart service
echo   -s status       Show service status
echo.
echo To run on Windows, use:
echo   iptv-aggregator.exe -port 8080
echo.
pause
