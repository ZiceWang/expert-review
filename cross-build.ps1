#Requires -Version 5.0

$ErrorActionPreference = "Stop"

$OrigDir = $PWD.Path
$ScriptDir = $PSScriptRoot
$ReleasesDir = Join-Path $ScriptDir "releases"
$McpServer = Join-Path $ScriptDir "mcp-server"
$Frontend = Join-Path $ScriptDir "frontend"
$Version = "0.0.1"

$Targets = @(
    @{OS="darwin"; ARCH="amd64"},
    @{OS="darwin"; ARCH="arm64"},
    @{OS="freebsd"; ARCH="amd64"},
    @{OS="linux"; ARCH="386"},
    @{OS="linux"; ARCH="amd64"},
    @{OS="linux"; ARCH="arm"},
    @{OS="linux"; ARCH="arm64"},
    @{OS="linux"; ARCH="riscv64"},
    @{OS="openbsd"; ARCH="amd64"},
    @{OS="windows"; ARCH="386"},
    @{OS="windows"; ARCH="amd64"},
    @{OS="windows"; ARCH="arm64"}
)

Write-Host "================================" -ForegroundColor Cyan
Write-Host "Cross-compile Expert Review MCP"
Write-Host "================================"
Write-Host ""

# Clean releases dir
if (Test-Path $ReleasesDir) {
    Remove-Item -Recurse -Force $ReleasesDir
}
New-Item -ItemType Directory -Path $ReleasesDir | Out-Null

# Build frontend once
Write-Host "[1/3] Building frontend..." -ForegroundColor Yellow
Set-Location $Frontend
npm install
npm run build

Write-Host ""
Write-Host "[2/3] Cross-compiling Go server..." -ForegroundColor Yellow

foreach ($Target in $Targets) {
    $OS = $Target.OS
    $ARCH = $Target.ARCH
    $Basename = "expert_review-${Version}-${OS}_${ARCH}"
    $OutputName = if ($OS -eq "windows") { "expert-review.exe" } else { "expert-review" }

    Write-Host "  Building $Basename..."

    # Build
    Set-Location $McpServer
    $env:GOOS = $OS
    $env:GOARCH = $ARCH
    go build -o "$ReleasesDir/$Basename/$OutputName" .

    # Copy public (frontend)
    $TargetDir = "$ReleasesDir/$Basename"
    New-Item -ItemType Directory -Path "$TargetDir/public" -Force | Out-Null
    Copy-Item -Path "$Frontend/dist/*" -Destination "$TargetDir/public/" -Recurse -Force

    # Create zip
    $ZipPath = "$ReleasesDir/$Basename.zip"
    if (Test-Path $ZipPath) {
        Remove-Item $ZipPath -Force
    }
    Compress-Archive -Path "$TargetDir/*" -DestinationPath $ZipPath -Force

    # Cleanup folder
    Remove-Item -Recurse -Force $TargetDir

    Write-Host "  -> ${Basename}.zip" -ForegroundColor Green
}

Write-Host ""
Write-Host "[3/3] Summary" -ForegroundColor Yellow
Write-Host "================================" -ForegroundColor Cyan
Get-ChildItem $ReleasesDir -Filter "*.zip" | ForEach-Object {
    $size = if ($_.Length -gt 1MB) { "{0:N1} MB" -f ($_.Length / 1MB) } else { "{0:N0} KB" -f ($_.Length / 1KB) }
    Write-Host "  $($_.Name) - $size"
}

Set-Location $OrigDir
Write-Host ""
Write-Host "Done: $ReleasesDir" -ForegroundColor Green
