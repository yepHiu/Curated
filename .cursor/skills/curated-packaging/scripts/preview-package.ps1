[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$Mode,

  [Parameter(Mandatory = $true)]
  [string]$CurrentBaseVersion,

  [string]$RequestedMajor,

  [string]$RequestedMinor
)

$majorValue = if ([string]::IsNullOrWhiteSpace($RequestedMajor)) {
  $null
} else {
  [int]$RequestedMajor
}

$minorValue = if ([string]::IsNullOrWhiteSpace($RequestedMinor)) {
  $null
} else {
  [int]$RequestedMinor
}

$currentParts = $CurrentBaseVersion.Split(".")
$major = if ($null -ne $majorValue) { $majorValue } else { [int]$currentParts[0] }
$minor = if ($null -ne $minorValue) { $minorValue } else { [int]$currentParts[1] }
$basePatch = if ($null -ne $majorValue -or $null -ne $minorValue) {
  0
} else {
  [int]$currentParts[2]
}

$willBumpPatch = $Mode -ne "preview" -and $Mode -ne "set-base"
$predictedPatch = if ($willBumpPatch) { $basePatch + 1 } else { $basePatch }

$result = [ordered]@{
  mode = $Mode
  currentBaseVersion = $CurrentBaseVersion
  baseVersionAfterChange = "$major.$minor.$basePatch"
  predictedVersion = "$major.$minor.$predictedPatch"
  willBumpPatch = $willBumpPatch
}

$result | ConvertTo-Json -Depth 5
