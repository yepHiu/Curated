[CmdletBinding()]
param(
  [string]$Version,

  [string]$BuildStamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd.HHmmss"),

  [string]$OutputDir = "release",

  [string]$VersionFile = "release/version.json",

  [string]$HistoryPath = "docs/2026-04-02-package-build-history.md"
)

$ErrorActionPreference = "Stop"

. (Join-Path $PSScriptRoot "release-common.ps1")
$repoRoot = Get-ReleaseRepoRoot -ScriptRoot $PSScriptRoot
$versionInfo = Resolve-ReleaseVersion -RepoRoot $repoRoot -Version $Version -VersionFile $VersionFile
$Version = $versionInfo.Version
$resolvedOutputDir = Resolve-ReleasePath -RepoRoot $repoRoot -PathValue $OutputDir
$manifestDir = Join-Path $resolvedOutputDir "manifest"
$portableZip = Join-Path (Join-Path $resolvedOutputDir "portable") ("Curated-{0}-windows-x64.zip" -f $Version)
$installerExe = Join-Path (Join-Path $resolvedOutputDir "installer") ("Curated-Setup-{0}.exe" -f $Version)
$assembledDir = Join-Path $resolvedOutputDir "Curated"
$binaryPath = Join-Path $resolvedOutputDir "backend\curated.exe"
$manifestPath = Join-Path $manifestDir "release.json"
$status = "failed"
$artifactPaths = @()
$notes = "versionSource=$($versionInfo.Source); BuildStamp=$BuildStamp"

try {
  Write-Host "==> Publishing Curated release"
  Write-Host "Version   : $Version"
  Write-Host "Source    : $($versionInfo.Source)"
  Write-Host "BuildStamp: $BuildStamp"
  Write-Host "Output    : $resolvedOutputDir"

  & (Join-Path $PSScriptRoot "build-frontend.ps1") -Version $Version -OutputDir (Join-Path $resolvedOutputDir "frontend")
  & (Join-Path $PSScriptRoot "build-backend.ps1") -Version $Version -BuildStamp $BuildStamp -OutputDir (Join-Path $resolvedOutputDir "backend")
  & (Join-Path $PSScriptRoot "assemble-release.ps1") -Version $Version -BuildStamp $BuildStamp -BinaryPath $binaryPath -FrontendDistDir (Join-Path $resolvedOutputDir "frontend") -OutputDir $assembledDir
  $portableResult = & (Join-Path $PSScriptRoot "package-portable.ps1") -Version $Version -InputDir $assembledDir -OutputDir (Join-Path $resolvedOutputDir "portable") -SkipHistory
  $installerResult = & (Join-Path $PSScriptRoot "package-installer.ps1") -Version $Version -AppDir $assembledDir -OutputDir (Join-Path $resolvedOutputDir "installer") -SkipHistory

  New-Item -ItemType Directory -Path $manifestDir -Force | Out-Null

  $manifest = [ordered]@{
    productName = "Curated"
    version = $Version
    buildStamp = $BuildStamp
    channel = "release"
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    artifacts = @()
  }

  if (Test-Path $portableZip) {
    $portableHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $portableZip).Hash
    $manifest.artifacts += [ordered]@{
      type = "portable"
      fileName = [System.IO.Path]::GetFileName($portableZip)
      path = $portableZip
      sha256 = $portableHash
    }
  }

  if (Test-Path $installerExe) {
    $installerHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $installerExe).Hash
    $manifest.artifacts += [ordered]@{
      type = "installer"
      fileName = [System.IO.Path]::GetFileName($installerExe)
      path = $installerExe
      sha256 = $installerHash
    }
  }

  $manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $manifestPath -Encoding utf8

  $artifactPaths = @()
  if ($portableResult -and $portableResult.ArtifactPaths) {
    $artifactPaths += [string[]]$portableResult.ArtifactPaths
  }
  if ($installerResult -and $installerResult.ArtifactPaths) {
    $artifactPaths += [string[]]$installerResult.ArtifactPaths
  }
  $artifactPaths += $manifestPath

  $resultStatuses = @()
  if ($portableResult) {
    $resultStatuses += [string]$portableResult.Status
  }
  if ($installerResult) {
    $resultStatuses += [string]$installerResult.Status
  }

  if ($resultStatuses -contains "failed") {
    $status = "failed"
  } elseif ($resultStatuses -contains "partial") {
    $status = "partial"
  } else {
    $status = "success"
  }

  if ($portableResult -and -not [string]::IsNullOrWhiteSpace($portableResult.Notes)) {
    $notes = "$notes; portable=$($portableResult.Notes)"
  }
  if ($installerResult -and -not [string]::IsNullOrWhiteSpace($installerResult.Notes)) {
    $notes = "$notes; installer=$($installerResult.Notes)"
  }

  Write-Host "Release publish flow complete."
}
catch {
  $notes = "$notes; error=$($_.Exception.Message)"
  throw
}
finally {
  if (-not [string]::IsNullOrWhiteSpace($Version)) {
    Add-PackageBuildHistoryEntry `
      -RepoRoot $repoRoot `
      -HistoryPath $HistoryPath `
      -Version $Version `
      -BuildType "release:publish" `
      -ArtifactPaths $artifactPaths `
      -Status $status `
      -Operator (Get-ReleaseOperator) `
      -Notes $notes
  }
}
