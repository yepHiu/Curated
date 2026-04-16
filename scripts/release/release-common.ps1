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

function Invoke-ReleasePackageHistoryTool {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [Parameter(Mandatory = $true)]
    [string[]]$Arguments
  )

  $toolPath = Join-Path $RepoRoot "scripts\release\package-history.mjs"
  $output = & node $toolPath @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "Package history tool failed with exit code $LASTEXITCODE"
  }

  return $output | ConvertFrom-Json
}

function Get-ReleaseLocalizedText {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet(
      "first-record",
      "prev-commit-unresolved",
      "current-commit-unresolved",
      "no-diff",
      "change-summary-failed-prefix",
      "status-success",
      "status-failed",
      "status-partial"
    )]
    [string]$Key
  )

  switch ($Key) {
    "first-record" { return [string]([char[]]@(0x9996, 0x6761, 0x6253, 0x5305, 0x8bb0, 0x5f55, 0xff0c, 0x65e0, 0x4e0a, 0x4e00, 0x5305, 0x53ef, 0x6bd4, 0x5bf9)) }
    "prev-commit-unresolved" { return [string]([char[]]@(0x65e0, 0x6cd5, 0x89e3, 0x6790, 0x4e0a, 0x4e00, 0x6761, 0x6253, 0x5305, 0x8bb0, 0x5f55, 0x5bf9, 0x5e94, 0x20, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74)) }
    "current-commit-unresolved" { return [string]([char[]]@(0x65e0, 0x6cd5, 0x89e3, 0x6790, 0x5f53, 0x524d, 0x6253, 0x5305, 0x8bb0, 0x5f55, 0x5bf9, 0x5e94, 0x20, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74)) }
    "no-diff" { return [string]([char[]]@(0x65e0, 0x4ee3, 0x7801, 0x5dee, 0x5f02, 0xff08, 0x540c, 0x4e00, 0x63d0, 0x4ea4, 0x91cd, 0x590d, 0x6253, 0x5305, 0xff09)) }
    "change-summary-failed-prefix" { return [string]([char[]]@(0x65e0, 0x6cd5, 0x751f, 0x6210, 0x53d8, 0x66f4, 0x5185, 0x5bb9, 0xff1a)) }
    "status-success" { return [string]([char[]]@(0x6210, 0x529f)) }
    "status-failed" { return [string]([char[]]@(0x5931, 0x8d25)) }
    "status-partial" { return [string]([char[]]@(0x90e8, 0x5206, 0x6210, 0x529f)) }
  }
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

function Get-PackageHistoryChangeSummary {
  param(
    [Parameter(Mandatory = $true)]
    [string]$RepoRoot,

    [Parameter(Mandatory = $true)]
    [string]$HistoryPath,

    [Parameter(Mandatory = $true)]
    [string]$CurrentCommit
  )

  $resolvedHistoryPath = Resolve-ReleasePath -RepoRoot $RepoRoot -PathValue $HistoryPath

  Push-Location $RepoRoot
  try {
    if (-not (Test-Path $resolvedHistoryPath)) {
      return (Get-ReleaseLocalizedText -Key "first-record")
    }

    $result = Invoke-ReleasePackageHistoryTool -RepoRoot $RepoRoot -Arguments @(
      "previous-commit",
      "--history-path",
      $resolvedHistoryPath
    )

    $previousCommit = ""
    if ($null -ne $result -and $result.PSObject.Properties.Match("commit").Count -gt 0) {
      $previousCommit = [string]$result.commit
    }

    if ([string]::IsNullOrWhiteSpace($previousCommit)) {
      return (Get-ReleaseLocalizedText -Key "first-record")
    }

    $previousResolved = (& git rev-parse --verify $previousCommit 2>$null).Trim()
    if ([string]::IsNullOrWhiteSpace($previousResolved)) {
      return (Get-ReleaseLocalizedText -Key "prev-commit-unresolved")
    }

    $currentResolved = (& git rev-parse --verify $CurrentCommit 2>$null).Trim()
    if ([string]::IsNullOrWhiteSpace($currentResolved)) {
      return (Get-ReleaseLocalizedText -Key "current-commit-unresolved")
    }

    if ($previousResolved -eq $currentResolved) {
      return (Get-ReleaseLocalizedText -Key "no-diff")
    }

    $gitRange = $previousResolved + ".." + $currentResolved
    $rawLogLines = @(& git log --oneline $gitRange 2>$null)
    $normalizedLogLines = @($rawLogLines | ForEach-Object { [string]$_ } | Where-Object {
      -not [string]::IsNullOrWhiteSpace($_)
    })

    if ($normalizedLogLines.Count -eq 0) {
      return (Get-ReleaseLocalizedText -Key "no-diff")
    }

    return ($normalizedLogLines -join "<br>")
  }
  catch {
    return ((Get-ReleaseLocalizedText -Key "change-summary-failed-prefix") + $_.Exception.Message)
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
  $changeSummary = Get-PackageHistoryChangeSummary -RepoRoot $RepoRoot -HistoryPath $HistoryPath -CurrentCommit $gitContext.Sha

  $normalizedArtifactPaths = @()
  foreach ($artifactPath in $ArtifactPaths) {
    if (-not [string]::IsNullOrWhiteSpace($artifactPath)) {
      $normalizedArtifactPaths += (Convert-ToRepoRelativePath -RepoRoot $RepoRoot -PathValue $artifactPath)
    }
  }

  switch ($Status) {
    "success" { $statusLabel = Get-ReleaseLocalizedText -Key "status-success" }
    "failed" { $statusLabel = Get-ReleaseLocalizedText -Key "status-failed" }
    "partial" { $statusLabel = Get-ReleaseLocalizedText -Key "status-partial" }
    default { $statusLabel = $Status }
  }

  $artifactCell = ($normalizedArtifactPaths -join "; ")
  $commitCell = "$($gitContext.Sha) / $($gitContext.Branch)"
  $dateCell = (Get-Date).ToString("yyyy-MM-dd")
  $entry = "| {0} | {1} | `{2}` | `{3}` | `{4}` | {5} | {6} | {7} | {8} |" -f `
    (Format-HistoryCell $dateCell), `
    (Format-HistoryCell $Version), `
    (Format-HistoryCell $commitCell), `
    (Format-HistoryCell $BuildType), `
    (Format-HistoryCell $artifactCell), `
    (Format-HistoryCell $statusLabel), `
    (Format-HistoryCell $Operator), `
    (Format-HistoryCell $changeSummary), `
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
