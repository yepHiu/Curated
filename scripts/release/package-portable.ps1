[CmdletBinding()]
param(
  [string]$Version,

  [string]$InputDir = "release/Curated",

  [string]$OutputDir = "release/portable",

  [string]$VersionFile = "release/version.json",

  [string]$HistoryPath = "docs/2026-04-02-package-build-history.md",

  [switch]$SkipHistory
)

$ErrorActionPreference = "Stop"

. (Join-Path $PSScriptRoot "release-common.ps1")
$repoRoot = Get-ReleaseRepoRoot -ScriptRoot $PSScriptRoot
$versionInfo = Resolve-ReleaseVersion -RepoRoot $repoRoot -Version $Version -VersionFile $VersionFile
$Version = $versionInfo.Version
$resolvedInputDir = Resolve-ReleasePath -RepoRoot $repoRoot -PathValue $InputDir
$resolvedOutputDir = Resolve-ReleasePath -RepoRoot $repoRoot -PathValue $OutputDir
$zipPath = Join-Path $resolvedOutputDir ("Curated-{0}-windows-x64.zip" -f $Version)
$status = "failed"
$artifactPaths = @()
$notes = "versionSource=$($versionInfo.Source)"

try {
  if (-not (Test-Path $resolvedInputDir)) {
    throw "Release directory not found: $resolvedInputDir"
  }

  Write-Host "==> Packaging portable zip"
  Write-Host "Version: $Version"
  Write-Host "Source : $($versionInfo.Source)"
  Write-Host "Input  : $resolvedInputDir"
  Write-Host "Output : $zipPath"

  New-Item -ItemType Directory -Path $resolvedOutputDir -Force | Out-Null
  if (Test-Path $zipPath) {
    Remove-Item -LiteralPath $zipPath -Force
  }

  Compress-Archive -Path (Join-Path $resolvedInputDir "*") -DestinationPath $zipPath -CompressionLevel Optimal

  $status = "success"
  $artifactPaths = @($zipPath)

  Write-Host "Portable zip created."

  return [pscustomobject]@{
    Version       = $Version
    Status        = $status
    ArtifactPaths = $artifactPaths
    Notes         = $notes
  }
}
catch {
  $notes = "$notes; error=$($_.Exception.Message)"
  throw
}
finally {
  if (-not $SkipHistory -and -not [string]::IsNullOrWhiteSpace($Version)) {
    Add-PackageBuildHistoryEntry `
      -RepoRoot $repoRoot `
      -HistoryPath $HistoryPath `
      -Version $Version `
      -BuildType "release:portable" `
      -ArtifactPaths $artifactPaths `
      -Status $status `
      -Operator (Get-ReleaseOperator) `
      -Notes $notes
  }
}
