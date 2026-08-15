# Open Engineering Hub CLI — Windows Installer Script
# Usage: irm https://install.openengineering.dev/windows.ps1 | iex

$ErrorActionPreference = "Stop"

$REPO = "open-engineering-hub/oeh-cli"
$BIN_DIR = "$HOME\.oeh\bin"
$EXE_PATH = "$BIN_DIR\oeh.exe"

Write-Host ""
Write-Host "  Open Engineering Hub CLI Installer (Windows)" -ForegroundColor Cyan
Write-Host "  ───────────────────────────────────────────" -ForegroundColor Gray
Write-Host ""

# Create directory
if (!(Test-Path $BIN_DIR)) {
    New-Item -ItemType Directory -Force -Path $BIN_DIR | Out-Null
}

# Determine Arch
$ARCH = if ([Environment]::Is64BitProcess) { "amd64" } else { "32bit" }
$DOWNLOAD_URL = "https://github.com/$REPO/releases/latest/download/oeh-windows-$ARCH.exe"

Write-Host "  → Downloading OEH CLI (windows/$ARCH)..." -ForegroundColor Yellow

try {
    Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $EXE_PATH -UseBasicParsing
} catch {
    Write-Host "  ! Release binary download failed. Falling back to Go install..." -ForegroundColor Yellow
    if (Get-Command go -ErrorAction SilentlyContinue) {
        go install github.com/$REPO@latest
        Write-Host "  ✓ Installed via go install" -ForegroundColor Green
        exit 0
    } else {
        Write-Host "  ❌ Failed to download binary and Go is not installed." -ForegroundColor Red
        exit 1
    }
}

Write-Host "  ✓ Saved binary to $EXE_PATH" -ForegroundColor Green

# Add to User PATH if not present
$USER_PATH = [Environment]::GetEnvironmentVariable("Path", "User")
if ($USER_PATH -notlike "*$BIN_DIR*") {
    Write-Host "  → Adding $BIN_DIR to User PATH..." -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("Path", "$USER_PATH;$BIN_DIR", "User")
    $env:Path += ";$BIN_DIR"
    Write-Host "  ✓ Added to PATH! Restart your terminal after installation." -ForegroundColor Green
}

Write-Host ""
Write-Host "  ✓ Installation complete!" -ForegroundColor Green
Write-Host "  Get started:" -ForegroundColor Cyan
Write-Host "    oeh login --token <YOUR_TOKEN>"
Write-Host "    oeh doctor"
Write-Host ""
