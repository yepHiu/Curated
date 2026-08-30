[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-GitText {
  param(
    [Parameter(Mandatory)]
    [string[]]$Arguments,
    [string]$WorkingDirectory
  )

  $previousErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  try {
    if ($WorkingDirectory) {
      $result = & git -C $WorkingDirectory @Arguments 2>$null
    } else {
      $result = & git @Arguments 2>$null
    }
    $gitExitCode = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }

  if ($gitExitCode -ne 0) {
    return $null
  }

  return ($result -join "`n").Trim()
}

function Get-WorktreeRecords {
  param(
    [Parameter(Mandatory)]
    [string]$RepositoryRoot
  )

  $raw = Invoke-GitText -Arguments @('worktree', 'list', '--porcelain')
  if (-not $raw) {
    return @()
  }

  $records = foreach ($block in ($raw -split "(?:`r?`n){2,}")) {
    $fields = $block -split "`r?`n"
    $path = ($fields | Where-Object { $_ -like 'worktree *' } | Select-Object -First 1) -replace '^worktree ', ''
    if (-not $path) {
      continue
    }

    $head = (($fields | Where-Object { $_ -like 'HEAD *' } | Select-Object -First 1) -replace '^HEAD ', '')
    $branchRef = (($fields | Where-Object { $_ -like 'branch *' } | Select-Object -First 1) -replace '^branch refs/heads/', '')
    $isDetached = [bool]($fields | Where-Object { $_ -eq 'detached' })
    $porcelainText = Invoke-GitText -Arguments @('status', '--porcelain=v1') -WorkingDirectory $path
    $porcelain = @($porcelainText -split "`r?`n" | Where-Object { $_ })
    $untracked = @($porcelain | Where-Object { $_.StartsWith('?? ') }).Count
    $changed = @($porcelain | Where-Object { -not $_.StartsWith('?? ') }).Count

    [pscustomobject]@{
      path = $path
      isMain = ($path -eq $RepositoryRoot)
      head = if ($head) { $head.Substring(0, [Math]::Min(8, $head.Length)) } else { $null }
      branch = if ($branchRef) { $branchRef } elseif ($isDetached) { $null } else { $null }
      detached = $isDetached
      changedFiles = $changed
      untrackedFiles = $untracked
      lastCommit = Invoke-GitText -Arguments @('log', '-1', '--date=iso-strict', '--pretty=format:%h|%ad|%s') -WorkingDirectory $path
    }
  }

  return @($records)
}

$repositoryRoot = (Invoke-GitText -Arguments @('rev-parse', '--show-toplevel'))
if (-not $repositoryRoot) {
  throw 'Run this collector inside a Git worktree.'
}

Set-Location -LiteralPath $repositoryRoot
$requirementsPath = Join-Path $repositoryRoot 'docs/prd/requirements.csv'
$versionPath = Join-Path $repositoryRoot 'scripts/release/version.json'
$requirements = @(Import-Csv -LiteralPath $requirementsPath)
$statusCounts = [ordered]@{}
foreach ($row in $requirements) {
  if (-not $statusCounts.Contains($row.status)) {
    $statusCounts[$row.status] = 0
  }
  $statusCounts[$row.status]++
}

$upstream = Invoke-GitText -Arguments @('rev-parse', '--abbrev-ref', '--symbolic-full-name', '@{upstream}')
$aheadBehind = Invoke-GitText -Arguments @('rev-list', '--left-right', '--count', 'master...HEAD')
$ahead = $null
$behind = $null
if ($aheadBehind -match '^(\d+)\s+(\d+)$') {
  $behind = [int]$Matches[1]
  $ahead = [int]$Matches[2]
}

$version = Get-Content -Raw -LiteralPath $versionPath | ConvertFrom-Json
$branch = Invoke-GitText -Arguments @('branch', '--show-current')
$statusPorcelainText = Invoke-GitText -Arguments @('status', '--porcelain=v1')
$statusPorcelain = @($statusPorcelainText -split "`r?`n" | Where-Object { $_ })
$recentCommitText = Invoke-GitText -Arguments @('log', 'master..HEAD', '--date=short', '--pretty=format:%h|%ad|%s', '-n', '12')

$snapshot = [ordered]@{
  collectedAt = (Get-Date).ToString('yyyy-MM-dd HH:mm:ss K')
  requirements = [ordered]@{
    total = $requirements.Count
    statusCounts = $statusCounts
    inProgress = @($requirements | Where-Object { $_.status -eq 'in_progress' } | Select-Object id, title, priority, status, progress, detail_doc, updated_at)
    ideas = @($requirements | Where-Object { $_.status -eq 'idea' } | Select-Object id, title, priority, detail_doc, updated_at)
    recentlyDelivered = @($requirements | Where-Object { $_.status -in @('implemented', 'verified') } | Sort-Object updated_at -Descending | Select-Object -First 12 id, title, priority, status, progress, updated_at)
  }
  git = [ordered]@{
    branch = $branch
    head = Invoke-GitText -Arguments @('rev-parse', '--short', 'HEAD')
    headDetail = Invoke-GitText -Arguments @('log', '-1', '--date=iso-strict', '--pretty=format:%h|%ad|%s')
    upstream = $upstream
    aheadOfMaster = $ahead
    behindMaster = $behind
    origin = Invoke-GitText -Arguments @('remote', 'get-url', 'origin')
    releaseVersion = "$($version.current.major).$($version.current.minor).$($version.current.patch)"
    statusPorcelain = $statusPorcelain
    recentCommits = @($recentCommitText -split "`r?`n" | Where-Object { $_ })
  }
  worktrees = Get-WorktreeRecords -RepositoryRoot $repositoryRoot
}

$snapshot | ConvertTo-Json -Depth 7
