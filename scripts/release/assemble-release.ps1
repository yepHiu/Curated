[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [Parameter(Mandatory = $true)]
  [string]$BuildStamp,

  [string]$BinaryPath = "release/backend/curated.exe",

  [string]$FrontendDistDir = "release/frontend",

  [string]$OutputDir = "release/Curated"
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
if ([System.IO.Path]::IsPathRooted($BinaryPath)) {
  $resolvedBinaryPath = [System.IO.Path]::GetFullPath($BinaryPath)
} else {
  $resolvedBinaryPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $BinaryPath))
}
if ([System.IO.Path]::IsPathRooted($FrontendDistDir)) {
  $resolvedFrontendDistDir = [System.IO.Path]::GetFullPath($FrontendDistDir)
} else {
  $resolvedFrontendDistDir = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $FrontendDistDir))
}
if ([System.IO.Path]::IsPathRooted($OutputDir)) {
  $resolvedOutputDir = [System.IO.Path]::GetFullPath($OutputDir)
} else {
  $resolvedOutputDir = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDir))
}
$runtimeDir = Join-Path $resolvedOutputDir "runtime"
$docsDir = Join-Path $resolvedOutputDir "docs"
$thirdPartyDir = Join-Path $repoRoot "backend\third_party"

if (-not (Test-Path $resolvedBinaryPath)) {
  throw "Release binary not found: $resolvedBinaryPath"
}

Write-Host "==> Assembling release directory"
Write-Host "Version : $Version"
Write-Host "Channel : release"
Write-Host "Output  : $resolvedOutputDir"

if (Test-Path $resolvedOutputDir) {
  Remove-Item -LiteralPath $resolvedOutputDir -Recurse -Force
}

New-Item -ItemType Directory -Path $resolvedOutputDir -Force | Out-Null
New-Item -ItemType Directory -Path $runtimeDir -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $runtimeDir "config") -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $runtimeDir "data") -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $runtimeDir "cache") -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $runtimeDir "logs") -Force | Out-Null
New-Item -ItemType Directory -Path $docsDir -Force | Out-Null

Copy-Item -LiteralPath $resolvedBinaryPath -Destination (Join-Path $resolvedOutputDir "curated.exe") -Force
Copy-Item -LiteralPath (Join-Path $repoRoot "backend\internal\assets\curated.ico") -Destination (Join-Path $resolvedOutputDir "curated.ico") -Force

if (Test-Path $resolvedFrontendDistDir) {
  Copy-Item -LiteralPath $resolvedFrontendDistDir -Destination (Join-Path $resolvedOutputDir "frontend-dist") -Recurse -Force
}

if (Test-Path $thirdPartyDir) {
  Copy-Item -LiteralPath $thirdPartyDir -Destination (Join-Path $resolvedOutputDir "third_party") -Recurse -Force
}

Copy-Item -LiteralPath (Join-Path $repoRoot "config\library-config.cfg") -Destination (Join-Path $runtimeDir "config\library-config.example.cfg") -Force
Copy-Item -LiteralPath (Join-Path $repoRoot "docs\plan\2026-03-31-production-packaging-and-config-strategy.md") -Destination (Join-Path $docsDir "production-packaging-and-config-strategy.md") -Force

$notes = @"
Curated release package

Version    : $Version
BuildStamp : $BuildStamp
Channel    : release

This package is prepared for both installer and portable distribution.

Runtime data layout:
  config\
  data\
  cache\
  logs\

Current status:
  - curated.exe is the release backend binary.
  - frontend-dist contains the production frontend output.
  - third_party can contain bundled runtime tools such as ffmpeg.
  - runtime\config\library-config.example.cfg is a sample library settings file.

Target production behavior:
  - release builds should use the per-user data directory by default.
  - config, database, cache, and logs should stay outside the install directory.
"@

Set-Content -LiteralPath (Join-Path $resolvedOutputDir "README-release.txt") -Value $notes -Encoding utf8

Write-Host "Release directory assembled."
