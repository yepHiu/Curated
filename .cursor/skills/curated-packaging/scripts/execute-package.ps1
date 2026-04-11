[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("publish", "installer", "portable", "set-base")]
  [string]$Mode,

  [Nullable[int]]$Major,

  [Nullable[int]]$Minor
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\..")).Path

Push-Location $repoRoot
try {
  switch ($Mode) {
    "publish" {
      & pnpm release:publish
      break
    }
    "installer" {
      & pnpm release:installer
      break
    }
    "portable" {
      & pnpm release:portable
      break
    }
    "set-base" {
      if ($null -eq $Major -or $null -eq $Minor) {
        throw "Mode set-base requires -Major and -Minor."
      }

      & pnpm release:version:set-base -- --Major $Major --Minor $Minor
      break
    }
  }

  if ($LASTEXITCODE -ne 0) {
    throw "Packaging command failed with exit code $LASTEXITCODE"
  }
}
finally {
  Pop-Location
}
