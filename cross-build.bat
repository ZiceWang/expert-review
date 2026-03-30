@echo off
setlocal enabledelayedexpansion

echo ================================
echo Cross-compile Expert Review MCP
echo ================================

set "PROJECT_ROOT=%~dp0"
set "RELEASES_DIR=%PROJECT_ROOT%releases"
set "MCP_SERVER=%PROJECT_ROOT%mcp-server"
set "FRONTEND=%PROJECT_ROOT%frontend"
set "VERSION=0.0.1"

:: Clean releases dir
if exist "%RELEASES_DIR%" rmdir /s /q "%RELEASES_DIR%"
mkdir "%RELEASES_DIR%"

echo.
echo [1/3] Building frontend...
cd /d "%FRONTEND%"
call npm install
call npm run build

echo.
echo [2/3] Cross-compiling Go server...

:: Darwin
call :build_target darwin amd64
call :build_target darwin arm64

:: FreeBSD
call :build_target freebsd amd64

:: Linux
call :build_target linux 386
call :build_target linux amd64
call :build_target linux arm
call :build_target linux arm64
call :build_target linux riscv64

:: OpenBSD
call :build_target openbsd amd64

:: Windows
call :build_target windows 386
call :build_target windows amd64
call :build_target windows arm64

echo.
echo [3/3] Summary
echo ================================
dir /b "%RELEASES_DIR%\*.zip" 2>nul
echo ================================

cd /d "%PROJECT_ROOT%"
echo Done: %RELEASES_DIR%
exit /b 0

:build_target
set OS=%1
set ARCH=%2
set BASENAME=expert_review-%VERSION%-%OS%_%ARCH%
set OUTPUT_NAME=expert-review
if "%OS%"=="windows" set OUTPUT_NAME=expert-review.exe

echo   Building %BASENAME%...

:: Build
cd /d "%MCP_SERVER%"
set GOOS=%OS%
set GOARCH=%ARCH%
go build -o "%RELEASES_DIR%\%BASENAME%\%OUTPUT_NAME%" .

:: Copy frontend
mkdir "%RELEASES_DIR%\%BASENAME%\public" 2>nul
xcopy /s /e /y "%FRONTEND%\dist\*" "%RELEASES_DIR%\%BASENAME%\public\" >nul 2>&1

:: Create zip
powershell -Command "Compress-Archive -Path '%RELEASES_DIR%\%BASENAME%\*' -DestinationPath '%RELEASES_DIR%\%BASENAME%.zip' -Force"

:: Cleanup folder
rmdir /s /q "%RELEASES_DIR%\%BASENAME%" 2>nul

echo   -^> %BASENAME%.zip
exit /b 0
