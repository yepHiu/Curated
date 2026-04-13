[CmdletBinding()]
param()

Set-StrictMode -Version Latest

function Get-ReleaseRepoRoot {
  param(
    [Parameter(Mandatory = $true)]
    [string]$ScriptRoot
  )

  return (Resolve-Path (Join-Path $ScriptRoot "..\..")).Path
}

function Resolve-ReleasePath {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [Parameter(Mandatory = $true)]
    [string]$PathValue
  )

  if ([System.IO.Path]::IsPathRooted($PathValue)) {
    return [System.IO.Path]::GetFullPath($PathValue)
  }

  return [System.IO.Path]::GetFullPath((Join-Path $RepoRoot $PathValue))
}

function Convert-ToRepoRelativePath {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [Parameter(Mandatory = $true)]
    [string]$PathValue
  )

  $resolvedPath = [System.IO.Path]::GetFullPath($PathValue)
  $getRelativePathMethod = [System.IO.Path].GetMethod(
    "GetRelativePath",
    [type[]]@([string], [string])
  )

  if ($null -ne $getRelativePathMethod) {
    $relative = [System.IO.Path]::GetRelativePath($RepoRoot, $resolvedPath)
  } else {
    $repoRootPath = [System.IO.Path]::GetFullPath($RepoRoot)
    if (
      -not $repoRootPath.EndsWith([System.IO.Path]::DirectorySeparatorChar) -and
      -not $repoRootPath.EndsWith([System.IO.Path]::AltDirectorySeparatorChar)
    ) {
      $repoRootPath += [System.IO.Path]::DirectorySeparatorChar
    }

    $repoRootUri = [System.Uri]::new($repoRootPath)
    $resolvedPathUri = [System.Uri]::new($resolvedPath)
    $relative = [System.Uri]::UnescapeDataString(
      $repoRootUri.MakeRelativeUri($resolvedPathUri).ToString()
    ).Replace("/", [System.IO.Path]::DirectorySeparatorChar)
  }

  return $relative.Replace("\", "/")
}

function Invoke-ReleaseVersionTool {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  $toolPath = Join-Path $RepoRoot "scripts\release\versioning.mjs"
  $output = & node $toolPath @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "Release version tool failed with exit code $LASTEXITCODE"
  }

  return $output | ConvertFrom-Json
}

function Resolve-ReleaseVersion {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [string]$Version,

    [string]$VersionFile = "scripts/release/version.json"
  )

  if (-not [string]::IsNullOrWhiteSpace($Version)) {
    return [pscustomobject]@{
      Version = $Version.Trim()
      Source  = "explicit"
    }
  }

  $versionFilePath = Resolve-ReleasePath -RepoRoot $RepoRoot -PathValue $VersionFile
  $result = Invoke-ReleaseVersionTool -RepoRoot $RepoRoot -Arguments @(
    "allocate",
    "--file",
    $versionFilePath
  )

  return [pscustomobject]@{
    Version = [string]$result.version
    Source  = "auto-patch"
  }
}

function Set-ReleaseVersionBase {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [Parameter(Mandatory = $true)]
    [int]$Major,

    [Parameter(Mandatory = $true)]
    [int]$Minor,

    [string]$VersionFile = "scripts/release/version.json"
  )

  $versionFilePath = Resolve-ReleasePath -RepoRoot $RepoRoot -PathValue $VersionFile
  $result = Invoke-ReleaseVersionTool -RepoRoot $RepoRoot -Arguments @(
    "set-base",
    "--file",
    $versionFilePath,
    "--major",
    $Major,
    "--minor",
    $Minor
  )

  return [pscustomobject]@{
    Version = [string]$result.version
    Source  = "manual-base"
  }
}

function Get-ReleaseGitContext {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot
  )

  Push-Location $RepoRoot
  try {
    $sha = (& git rev-parse --short HEAD 2>$null).Trim()
    if ([string]::IsNullOrWhiteSpace($sha)) {
      $sha = "unknown"
    }

    $branch = (& git branch --show-current 2>$null).Trim()
    if ([string]::IsNullOrWhiteSpace($branch)) {
      $branch = (& git rev-parse --abbrev-ref HEAD 2>$null).Trim()
    }
    if ([string]::IsNullOrWhiteSpace($branch)) {
      $branch = "unknown"
    }

    return [pscustomobject]@{
      Sha    = $sha
      Branch = $branch
    }
  }
  finally {
    Pop-Location
  }
}

function Format-HistoryCell {
  param(
    [AllowNull()]
    [AllowEmptyString()]
    [string]$Value
  )

  if ($null -eq $Value) {
    return ""
  }

  return $Value.Replace("|", "\|").Replace("`r", "").Replace("`n", "<br>")
}

function Add-PackageBuildHistoryEntry {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [Parameter(Mandatory = $true)]
    [string]$HistoryPath,

    [Parameter(Mandatory = $true)]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [string]$BuildType,

    [Parameter(Mandatory = $true)]
    [string[]]$ArtifactPaths,

    [Parameter(Mandatory = $true)]
    [string]$Status,

    [Parameter(Mandatory = $true)]
    [string]$Operator,

    [string]$Notes
  )

  $resolvedHistoryPath = Resolve-ReleasePath -RepoRoot $RepoRoot -PathValue $HistoryPath
  $gitContext = Get-ReleaseGitContext -RepoRoot $RepoRoot

  $normalizedArtifactPaths = @()
  foreach ($artifactPath in $ArtifactPaths) {
    if (-not [string]::IsNullOrWhiteSpace($artifactPath)) {
      $normalizedArtifactPaths += (Convert-ToRepoRelativePath -RepoRoot $RepoRoot -PathValue $artifactPath)
    }
  }

  switch ($Status) {
    "success" { $statusLabel = "成功" }
    "failed" { $statusLabel = "失败" }
    "partial" { $statusLabel = "部分成功" }
    default { $statusLabel = $Status }
  }

  $artifactCell = ($normalizedArtifactPaths -join "; ")
  $commitCell = "$($gitContext.Sha) / $($gitContext.Branch)"
  $dateCell = (Get-Date).ToString("yyyy-MM-dd")
  $entry = "| {0} | {1} | `{2}` | `{3}` | `{4}` | {5} | {6} | {7} |" -f `
    (Format-HistoryCell $dateCell), `
    (Format-HistoryCell $Version), `
    (Format-HistoryCell $commitCell), `
    (Format-HistoryCell $BuildType), `
    (Format-HistoryCell $artifactCell), `
    (Format-HistoryCell $statusLabel), `
    (Format-HistoryCell $Operator), `
    (Format-HistoryCell $Notes)

  Add-Content -LiteralPath $resolvedHistoryPath -Value $entry -Encoding utf8
}

function Get-ReleaseOperator {
  if (-not [string]::IsNullOrWhiteSpace($env:USERNAME)) {
    return $env:USERNAME
  }

  if (-not [string]::IsNullOrWhiteSpace($env:USER)) {
    return $env:USER
  }

  return "unknown"
}
