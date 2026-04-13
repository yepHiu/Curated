[CmdletBinding()]
param(
  [string]$Version,

  [string]$BuildStamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd.HHmmss"),

  [string]$OutputDir = "release",

  [string]$VersionFile = "scripts/release/version.json",

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

function Get-ReleaseResultProperty {
  param(
    [Parameter(Mandatory = $false)]
    [object]$Result,

    [Parameter(Mandatory = $true)]
    [string]$PropertyName
  )

  $resultItems = @(@($Result) | Where-Object { $null -ne $_ })
  for ($index = $resultItems.Count - 1; $index -ge 0; $index -= 1) {
    $candidate = $resultItems[$index]
    if ($candidate.PSObject.Properties.Match($PropertyName).Count -gt 0) {
      return $candidate.$PropertyName
    }
  }

  return $null
}

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
  $portableArtifactPaths = Get-ReleaseResultProperty -Result $portableResult -PropertyName "ArtifactPaths"
  if ($portableArtifactPaths) {
    $artifactPaths += [string[]]$portableArtifactPaths
  }
  $installerArtifactPaths = Get-ReleaseResultProperty -Result $installerResult -PropertyName "ArtifactPaths"
  if ($installerArtifactPaths) {
    $artifactPaths += [string[]]$installerArtifactPaths
  }
  $artifactPaths += $manifestPath

  $resultStatuses = @()
  $portableStatus = Get-ReleaseResultProperty -Result $portableResult -PropertyName "Status"
  if ($portableStatus) {
    $resultStatuses += [string]$portableStatus
  }
  $installerStatus = Get-ReleaseResultProperty -Result $installerResult -PropertyName "Status"
  if ($installerStatus) {
    $resultStatuses += [string]$installerStatus
  }

  if ($resultStatuses -contains "failed") {
    $status = "failed"
  } elseif ($resultStatuses -contains "partial") {
    $status = "partial"
  } else {
    $status = "success"
  }

  $portableNotes = Get-ReleaseResultProperty -Result $portableResult -PropertyName "Notes"
  if (-not [string]::IsNullOrWhiteSpace($portableNotes)) {
    $notes = "$notes; portable=$portableNotes"
  }
  $installerNotes = Get-ReleaseResultProperty -Result $installerResult -PropertyName "Notes"
  if (-not [string]::IsNullOrWhiteSpace($installerNotes)) {
    $notes = "$notes; installer=$installerNotes"
  }

  Write-Host "Release publish flow complete."
}
catch {
  $notes = "$notes; error=$($_.Exception.Message)"
  throw
}
finally {
  if (-not [string]::IsNullOrWhiteSpace($Version)) {
    $historyArtifactPaths = @($artifactPaths | Where-Object {
      -not [string]::IsNullOrWhiteSpace([string]$_)
    })

    if ($historyArtifactPaths.Count -eq 0) {
      if (Test-Path $portableZip) {
        $historyArtifactPaths += $portableZip
      }
      if (Test-Path $installerExe) {
        $historyArtifactPaths += $installerExe
      }
      if (Test-Path $manifestPath) {
        $historyArtifactPaths += $manifestPath
      }
    }

    if ($historyArtifactPaths.Count -gt 0) {
      Add-PackageBuildHistoryEntry `
        -RepoRoot $repoRoot `
        -HistoryPath $HistoryPath `
        -Version $Version `
        -BuildType "release:publish" `
        -ArtifactPaths $historyArtifactPaths `
        -Status $status `
        -Operator (Get-ReleaseOperator) `
        -Notes $notes
    }
  }
}
