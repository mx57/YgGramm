# File: scripts/build_windows.ps1
<#
.SYNOPSIS
    Builds the YggGram Go core into ygggram.dll for Windows (x64 / ARM64).
.DESCRIPTION
    Requires Go 1.23+ and a C compiler (GCC from MinGW-w64 or Clang).
    Outputs ygggram.dll and ygggram.h to the build folder and Flutter runner assets.
#>

[CmdletBinding()]
param(
    [ValidateSet("x64", "arm64")]
    [string]$Arch = "x64",
    [switch]$Release = $true,
    [string]$OutputDir = ""
)

$ErrorActionPreference = "Stop"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  YggGram Windows Core Compiler (Go C-Shared DLL)" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# Determine project root
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Resolve-Path (Join-Path $ScriptDir "..")
Set-Location $ProjectRoot

# Determine Target Go Architecture
$GoArch = if ($Arch -eq "x64") { "amd64" } else { "arm64" }
Write-Host "[*] Target Architecture: Windows $Arch (GOARCH=$GoArch)" -ForegroundColor Yellow

# Check Go toolchain
try {
    $GoVersion = & go version
    Write-Host "[*] Detected Go: $GoVersion" -ForegroundColor Gray
} catch {
    Write-Error "Go compiler not found. Please install Go 1.23+ and add it to PATH."
}

# Check C compiler for Cgo
$CCompiler = Get-Command "gcc" -ErrorAction SilentlyContinue
if (-not $CCompiler) {
    $CCompiler = Get-Command "clang" -ErrorAction SilentlyContinue
}

if (-not $CCompiler) {
    Write-Error "No C compiler (gcc or clang from MinGW-w64) found in PATH. CGO requires a working C toolchain."
} else {
    Write-Host "[*] Detected C Compiler: $($CCompiler.Source)" -ForegroundColor Gray
}

# Configure Output Path
if (-not $OutputDir) {
    $OutputDir = Join-Path $ProjectRoot "build\windows\$Arch"
}
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

$DllOutput = Join-Path $OutputDir "ygggram.dll"
$CorePkg = "./core/cmd/libygggram"

# Compiler Flags
$LdFlags = "-s -w -X main.BuildTime=$(Get-Date -Format 'yyyyMMdd-HHmmss')"
if ($Release) {
    $LdFlags += " -X main.Environment=production"
}

Write-Host "[*] Building $DllOutput..." -ForegroundColor Cyan

# Set Cgo & Go environment
$env:GOOS = "windows"
$env:GOARCH = $GoArch
$env:CGO_ENABLED = "1"

# Execute Build
& go build -buildmode=c-shared -ldflags "$LdFlags" -o "$DllOutput" $CorePkg

if ($LASTEXITCODE -ne 0) {
    Write-Error "Compilation failed with exit code $LASTEXITCODE"
}

Write-Host "[✓] Successfully built $DllOutput" -ForegroundColor Green

# Deploy to Flutter Windows runner directory if it exists
$FlutterWindowsTarget = Join-Path $ProjectRoot "app\windows\runner"
if (Test-Path $FlutterWindowsTarget) {
    Copy-Item -Path $DllOutput -Destination $FlutterWindowsTarget -Force
    Write-Host "[✓] Deployed DLL to Flutter Windows Runner: $FlutterWindowsTarget" -ForegroundColor Green
}
