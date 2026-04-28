param(
    [int]$BackendPort = 8080,
    [int]$FrontendPort = 5173,
    [switch]$Restart
)

$ErrorActionPreference = "Stop"

function Get-RepoRoot {
    $scriptPath = $PSCommandPath
    if (-not $scriptPath) {
        $scriptPath = $MyInvocation.MyCommand.Path
    }
    return (Resolve-Path (Join-Path (Split-Path -Parent $scriptPath) "..\..\..\..")).Path
}

function Get-ListenConnection {
    param([int]$Port)
    return Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
}

function Get-ProcessInfo {
    param([int]$ProcessId)
    $proc = Get-CimInstance Win32_Process -Filter "ProcessId = $ProcessId" -ErrorAction SilentlyContinue
    if (-not $proc) {
        return [PSCustomObject]@{ ProcessId = $ProcessId; Name = ""; CommandLine = "" }
    }
    return $proc
}

function Test-HttpOk {
    param(
        [string]$Url,
        [int]$TimeoutSec = 3
    )
    try {
        $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec $TimeoutSec
        return $response.StatusCode -ge 200 -and $response.StatusCode -lt 500
    }
    catch {
        return $false
    }
}

function Get-BackendHealth {
    param([int]$Port)
    try {
        return Invoke-RestMethod -Uri "http://127.0.0.1:$Port/api/health" -TimeoutSec 3
    }
    catch {
        return $null
    }
}

function Wait-Until {
    param(
        [scriptblock]$Condition,
        [int]$TimeoutSec,
        [string]$WaitingFor
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (& $Condition) {
            return $true
        }
        Start-Sleep -Milliseconds 500
    }
    throw "Timed out waiting for $WaitingFor"
}

function Ensure-WebApiEnv {
    param([string]$RepoRoot)

    $envPath = Join-Path $RepoRoot ".env"
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    if (-not (Test-Path $envPath)) {
        [System.IO.File]::WriteAllLines($envPath, @("VITE_USE_WEB_API=true"), $utf8NoBom)
        return
    }

    $lines = Get-Content -Path $envPath
    $found = $false
    $updated = foreach ($line in $lines) {
        if ($line -match "^\s*VITE_USE_WEB_API\s*=") {
            $found = $true
            "VITE_USE_WEB_API=true"
        }
        else {
            $line
        }
    }
    if (-not $found) {
        $updated += "VITE_USE_WEB_API=true"
    }
    [System.IO.File]::WriteAllLines($envPath, $updated, $utf8NoBom)
}

function Stop-ExpectedPortOwner {
    param(
        [int]$Port,
        [string[]]$ExpectedNames,
        [string]$ExpectedCommandPattern
    )

    $connection = Get-ListenConnection -Port $Port
    if (-not $connection) {
        return
    }

    $owner = Get-ProcessInfo -ProcessId $connection.OwningProcess
    $nameMatches = $ExpectedNames -contains $owner.Name
    $commandMatches = $owner.CommandLine -match $ExpectedCommandPattern
    if (-not ($nameMatches -or $commandMatches)) {
        throw "Port $Port is occupied by PID $($owner.ProcessId) ($($owner.Name)): $($owner.CommandLine)"
    }

    Stop-Process -Id $connection.OwningProcess -Force
    Start-Sleep -Milliseconds 500
}

$repoRoot = Get-RepoRoot
$backendRoot = Join-Path $repoRoot "backend"
$logDir = Join-Path $repoRoot ".workspace\dev-logs"
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

Ensure-WebApiEnv -RepoRoot $repoRoot

if ($Restart) {
    Stop-ExpectedPortOwner -Port $FrontendPort -ExpectedNames @("node.exe", "node") -ExpectedCommandPattern "vite|pnpm"
    Stop-ExpectedPortOwner -Port $BackendPort -ExpectedNames @("curated.exe", "curated") -ExpectedCommandPattern "cmd/curated|curated"
}

$backendHealth = Get-BackendHealth -Port $BackendPort
$backendStarted = $false
if (-not $backendHealth) {
    $backendConnection = Get-ListenConnection -Port $BackendPort
    if ($backendConnection) {
        $owner = Get-ProcessInfo -ProcessId $backendConnection.OwningProcess
        throw "Backend port $BackendPort is occupied but /api/health did not respond. PID $($owner.ProcessId) ($($owner.Name)): $($owner.CommandLine)"
    }

    $backendOut = Join-Path $logDir "backend.out.log"
    $backendErr = Join-Path $logDir "backend.err.log"
    Start-Process -FilePath "powershell" `
        -ArgumentList @("-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "go run ./cmd/curated") `
        -WorkingDirectory $backendRoot `
        -WindowStyle Hidden `
        -RedirectStandardOutput $backendOut `
        -RedirectStandardError $backendErr | Out-Null
    $backendStarted = $true
    Wait-Until -TimeoutSec 45 -WaitingFor "Curated backend on port $BackendPort" -Condition {
        $null -ne (Get-BackendHealth -Port $BackendPort)
    } | Out-Null
    $backendHealth = Get-BackendHealth -Port $BackendPort
}

$frontendStarted = $false
if (-not (Test-HttpOk -Url "http://127.0.0.1:$FrontendPort/" -TimeoutSec 3)) {
    $frontendConnection = Get-ListenConnection -Port $FrontendPort
    if ($frontendConnection) {
        $owner = Get-ProcessInfo -ProcessId $frontendConnection.OwningProcess
        throw "Frontend port $FrontendPort is occupied but did not respond. PID $($owner.ProcessId) ($($owner.Name)): $($owner.CommandLine)"
    }

    $frontendOut = Join-Path $logDir "frontend.out.log"
    $frontendErr = Join-Path $logDir "frontend.err.log"
    Start-Process -FilePath "powershell" `
        -ArgumentList @("-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "pnpm dev -- --host 127.0.0.1 --port $FrontendPort") `
        -WorkingDirectory $repoRoot `
        -WindowStyle Hidden `
        -RedirectStandardOutput $frontendOut `
        -RedirectStandardError $frontendErr | Out-Null
    $frontendStarted = $true
    Wait-Until -TimeoutSec 30 -WaitingFor "Vite frontend on port $FrontendPort" -Condition {
        Test-HttpOk -Url "http://127.0.0.1:$FrontendPort/" -TimeoutSec 3
    } | Out-Null
}

$backendConnectionFinal = Get-ListenConnection -Port $BackendPort
$frontendConnectionFinal = Get-ListenConnection -Port $FrontendPort

[PSCustomObject]@{
    BackendUrl = "http://127.0.0.1:$BackendPort"
    FrontendUrl = "http://127.0.0.1:$FrontendPort/"
    BackendStarted = $backendStarted
    FrontendStarted = $frontendStarted
    BackendPid = if ($backendConnectionFinal) { $backendConnectionFinal.OwningProcess } else { $null }
    FrontendPid = if ($frontendConnectionFinal) { $frontendConnectionFinal.OwningProcess } else { $null }
    BackendHealth = if ($backendHealth) { $backendHealth.name } else { $null }
    LogDir = $logDir
}
