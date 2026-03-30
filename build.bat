@echo off
setlocal enabledelayedexpansion

echo ================================
echo Expert Review MCP Server Build
echo ================================

set "PROJECT_ROOT=%~dp0"
set "BUILD_DIR=%PROJECT_ROOT%build"
set "MCP_SERVER=%PROJECT_ROOT%mcp-server"
set "FRONTEND=%PROJECT_ROOT%frontend"

:: Clean and create build dir
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
mkdir "%BUILD_DIR%\public"

echo.
echo [1/4] Installing Go dependencies...
cd /d "%MCP_SERVER%"
go mod tidy
if errorlevel 1 (
    echo FAILED: go mod tidy
    exit /b 1
)

echo.
echo [2/4] Building Go server...
cd /d "%MCP_SERVER%"
go build -o "%BUILD_DIR%\expert-review.exe" .
if errorlevel 1 (
    echo FAILED: go build
    exit /b 1
)

echo.
echo [3/4] Installing frontend dependencies...
cd /d "%FRONTEND%"
call npm install
if errorlevel 1 (
    echo FAILED: npm install
    exit /b 1
)

echo.
echo [4/4] Building Vue frontend...
call npm run build
if errorlevel 1 (
    echo FAILED: npm run build
    exit /b 1
)

echo.
echo [5/5] Copying frontend to build/public...
xcopy /s /e /y "%FRONTEND%\dist\*" "%BUILD_DIR%\public\" >nul 2>&1

echo.
echo ================================
echo Build complete!
echo ================================
echo.
echo Output: %BUILD_DIR%
echo   - expert-review.exe    (Go server)
echo   - public/            (Vue frontend)
echo.
echo To run:
echo   cd %BUILD_DIR%
echo   .\expert-review.exe
echo.
echo Frontend will be available at:
echo   http://localhost:3100
echo ================================

cd /d "%PROJECT_ROOT%"
