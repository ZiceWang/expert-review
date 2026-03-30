#Requires -Version 5.0

$ErrorActionPreference = "Stop"

$OrigDir = $PWD.Path
$ScriptDir = $PSScriptRoot
$BuildDir = Join-Path $ScriptDir "build"
$McpServer = Join-Path $ScriptDir "mcp-server"
$Frontend = Join-Path $ScriptDir "frontend"

Write-Host "================================" -ForegroundColor Cyan
Write-Host "Expert Review MCP Server Build"
Write-Host "================================"
Write-Host ""

# Clean and create build dir
if (Test-Path $BuildDir) {
    Remove-Item -Recurse -Force $BuildDir
}
New-Item -ItemType Directory -Path "$BuildDir/public" | Out-Null

Write-Host "[1/4] Go dependencies..." -ForegroundColor Yellow
Set-Location $McpServer
go mod tidy

Write-Host ""
Write-Host "[2/4] Building Go server..." -ForegroundColor Yellow
Set-Location $McpServer
go build -o "$BuildDir/expert-review.exe" .

Write-Host ""
Write-Host "[3/4] Frontend dependencies..." -ForegroundColor Yellow
Set-Location $Frontend
npm install

Write-Host ""
Write-Host "[4/4] Building Vue frontend..." -ForegroundColor Yellow
Set-Location $Frontend
npm run build

Write-Host ""
Write-Host "[5/5] Copying frontend to build/public..." -ForegroundColor Yellow
Copy-Item -Path "$Frontend/dist/*" -Destination "$BuildDir/public/" -Recurse -Force

Write-Host ""
Write-Host "================================" -ForegroundColor Cyan
Write-Host "Build complete!" -ForegroundColor Green
Write-Host "================================"
Write-Host ""
Write-Host "Output: $BuildDir/"
Write-Host "  - expert-review.exe    (Go server)"
Write-Host "  - public/            (Vue frontend)"
Write-Host ""
Write-Host "To run:"
Write-Host "  cd $BuildDir"
Write-Host "  .\expert-review.exe"
Write-Host ""
Write-Host "Frontend: http://localhost:3100"
Write-Host "================================"

Set-Location $OrigDir
