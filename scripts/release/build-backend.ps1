[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [string]$BuildStamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd.HHmmss"),

  [string]$OutputDir = "release/backend",

  [string]$BinaryName = "curated.exe"
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$backendRoot = Join-Path $repoRoot "backend"
$goCacheDir = Join-Path $repoRoot ".gocache"
if ([System.IO.Path]::IsPathRooted($OutputDir)) {
  $resolvedOutputDir = [System.IO.Path]::GetFullPath($OutputDir)
} else {
  $resolvedOutputDir = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDir))
}
$binaryPath = Join-Path $resolvedOutputDir $BinaryName

Write-Host "==> Building backend release binary"
Write-Host "Version   : $Version"
Write-Host "BuildStamp: $BuildStamp"
Write-Host "Output    : $binaryPath"

if (Test-Path $resolvedOutputDir) {
  Remove-Item -LiteralPath $resolvedOutputDir -Recurse -Force
}
New-Item -ItemType Directory -Path $resolvedOutputDir -Force | Out-Null
New-Item -ItemType Directory -Path $goCacheDir -Force | Out-Null

$ldflags = "-H=windowsgui -X curated-backend/internal/version.BuildStamp=$BuildStamp -X curated-backend/internal/version.InstallerVersion=$Version"

Push-Location $backendRoot
try {
  $env:GOCACHE = $goCacheDir
  go build -tags release -ldflags $ldflags -o $binaryPath ./cmd/curated
  if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
  }
}
finally {
  Pop-Location
}

Write-Host "Backend release build complete."
