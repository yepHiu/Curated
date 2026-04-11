[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [int]$Major,

  [Parameter(Mandatory = $true)]
  [int]$Minor,

  [string]$VersionFile = "release/version.json"
)

$ErrorActionPreference = "Stop"

. (Join-Path $PSScriptRoot "release-common.ps1")
$repoRoot = Get-ReleaseRepoRoot -ScriptRoot $PSScriptRoot
$result = Set-ReleaseVersionBase -RepoRoot $repoRoot -Major $Major -Minor $Minor -VersionFile $VersionFile

Write-Host "Release version base updated."
Write-Host "Version: $($result.Version)"
Write-Host "Source : $($result.Source)"
