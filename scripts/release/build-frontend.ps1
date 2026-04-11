[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [string]$OutputDir = "release/frontend"
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
if ([System.IO.Path]::IsPathRooted($OutputDir)) {
  $resolvedOutputDir = [System.IO.Path]::GetFullPath($OutputDir)
} else {
  $resolvedOutputDir = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDir))
}

Write-Host "==> Building frontend dist"
Write-Host "Version: $Version"
Write-Host "Output : $resolvedOutputDir"

Push-Location $repoRoot
try {
  $env:VITE_APP_VERSION = $Version
  pnpm typecheck
  if ($LASTEXITCODE -ne 0) {
    throw "pnpm typecheck failed with exit code $LASTEXITCODE"
  }

  pnpm exec vite build --configLoader native
  if ($LASTEXITCODE -ne 0) {
    throw "pnpm build failed with exit code $LASTEXITCODE"
  }

  if (Test-Path $resolvedOutputDir) {
    Remove-Item -LiteralPath $resolvedOutputDir -Recurse -Force
  }
  New-Item -ItemType Directory -Path $resolvedOutputDir -Force | Out-Null
  Copy-Item -Path (Join-Path $repoRoot "dist\*") -Destination $resolvedOutputDir -Recurse -Force
}
finally {
  Pop-Location
}

Write-Host "Frontend build complete."
