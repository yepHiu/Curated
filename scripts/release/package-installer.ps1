[CmdletBinding()]
param(
  [string]$Version,

  [string]$AppDir = "release/Curated",

  [string]$OutputDir = "release/installer",

  [string]$TemplatePath = "scripts/release/windows/Curated.iss.tpl",

  [string]$VersionFile = "scripts/release/version.json",

  [string]$HistoryPath = "docs/2026-04-02-package-build-history.md",

  [switch]$SkipHistory
)

$ErrorActionPreference = "Stop"

. (Join-Path $PSScriptRoot "release-common.ps1")
$repoRoot = Get-ReleaseRepoRoot -ScriptRoot $PSScriptRoot
$versionInfo = Resolve-ReleaseVersion -RepoRoot $repoRoot -Version $Version -VersionFile $VersionFile
$Version = $versionInfo.Version
$resolvedAppDir = Resolve-ReleasePath -RepoRoot $repoRoot -PathValue $AppDir
$resolvedOutputDir = Resolve-ReleasePath -RepoRoot $repoRoot -PathValue $OutputDir
$resolvedTemplatePath = Resolve-ReleasePath -RepoRoot $repoRoot -PathValue $TemplatePath
$generatedIssPath = Join-Path $resolvedOutputDir "Curated.iss"
$setupBaseName = "Curated-Setup-$Version"
$status = "failed"
$artifactPaths = @()
$notes = "versionSource=$($versionInfo.Source)"

try {
  if (-not (Test-Path $resolvedAppDir)) {
    throw "Release directory not found: $resolvedAppDir"
  }
  if (-not (Test-Path $resolvedTemplatePath)) {
    throw "Installer template not found: $resolvedTemplatePath"
  }

  Write-Host "==> Packaging installer"
  Write-Host "Version: $Version"
  Write-Host "Source : $($versionInfo.Source)"
  Write-Host "App dir: $resolvedAppDir"
  Write-Host "Output : $resolvedOutputDir"

  New-Item -ItemType Directory -Path $resolvedOutputDir -Force | Out-Null

  $template = Get-Content -LiteralPath $resolvedTemplatePath -Raw -Encoding utf8
  $template = $template.Replace("__APP_VERSION__", $Version)
  $template = $template.Replace("__APP_DIR__", $resolvedAppDir.Replace("\", "\\"))
  $template = $template.Replace("__OUTPUT_DIR__", $resolvedOutputDir.Replace("\", "\\"))
  $template = $template.Replace("__SETUP_BASENAME__", $setupBaseName)

  Set-Content -LiteralPath $generatedIssPath -Value $template -Encoding utf8

  $iscc = Get-Command "ISCC.exe" -ErrorAction SilentlyContinue
  if (-not $iscc) {
    $fallbacks = @(
      "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
      "C:\Program Files\Inno Setup 6\ISCC.exe"
    )
    foreach ($candidate in $fallbacks) {
      if (Test-Path $candidate) {
        $iscc = [pscustomobject]@{ Source = $candidate }
        break
      }
    }
  }
  if (-not $iscc) {
    Write-Warning "ISCC.exe not found. Generated installer script only: $generatedIssPath"
    Write-Warning "Install Inno Setup and rerun this script to build the installer."
    $status = "partial"
    $artifactPaths = @($generatedIssPath)
    $notes = "$notes; ISCC.exe not found, installer executable not generated"
    return [pscustomobject]@{
      Version       = $Version
      Status        = $status
      ArtifactPaths = $artifactPaths
      Notes         = $notes
    }
  }

  & $iscc.Source $generatedIssPath
  if ($LASTEXITCODE -ne 0) {
    throw "ISCC failed with exit code $LASTEXITCODE"
  }

  $status = "success"
  $artifactPaths = @(Join-Path $resolvedOutputDir "$setupBaseName.exe")

  Write-Host "Installer build complete."

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
      -BuildType "release:installer" `
      -ArtifactPaths $artifactPaths `
      -Status $status `
      -Operator (Get-ReleaseOperator) `
      -Notes $notes
  }
}
