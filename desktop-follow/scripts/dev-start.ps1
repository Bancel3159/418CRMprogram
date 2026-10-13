<#
.SYNOPSIS
  One-click dev starter for desktop-follow (CRM 维护端).
.NOTES
  - Vite & Tauri exe are started with -WindowStyle Hidden so no console window is visible.
  - crm-desktop-follow.exe has #![windows_subsystem = "windows"] baked in (see main.rs)
    so even without -WindowStyle Hidden there would be no console window.
  - Both children are fully detached: closing this script's terminal does not kill them.
    Use scripts\dev-stop.ps1 to kill them manually.
  - Flow:
    1. Kill leftovers on port 5275 + crm-desktop-follow.exe
    2. Background-start pnpm dev (logs in .dev-logs/vite.log)
    3. Poll localhost:5275 with TCP until 200 (timeout 30s)
    4. Start src-tauri\target\debug\crm-desktop-follow.exe
  Press Ctrl+C in this window to stop the watcher (children stay alive).
#>

$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

$VitePort   = 5275
$ExePath    = Join-Path $ProjectRoot 'src-tauri\target\debug\crm-desktop-follow.exe'
$ExeName    = 'crm-desktop-follow'
$LogDir     = Join-Path $ProjectRoot '.dev-logs'
if (-not (Test-Path $LogDir)) { New-Item -ItemType Directory -Path $LogDir | Out-Null }

function Write-Step($msg) {
  Write-Host ("`n[" + (Get-Date -Format 'HH:mm:ss') + "] " + $msg) -ForegroundColor Cyan
}

function Test-PortOpen {
  param([int]$Port, [int]$TimeoutMs = 1500)
  $client = New-Object System.Net.Sockets.TcpClient
  try {
    $iar = $client.BeginConnect('127.0.0.1', $Port, $null, $null)
    if (-not $iar.AsyncWaitHandle.WaitOne($TimeoutMs)) { return $false }
    $client.EndConnect($iar)
    return $true
  } catch {
    return $false
  } finally {
    $client.Close()
  }
}

function Stop-Occupying($port) {
  Write-Step ("Cleaning port " + $port)
  $conns = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
  foreach ($c in $conns) {
    $pid = $c.OwningProcess
    try {
      $p = Get-Process -Id $pid -ErrorAction SilentlyContinue
      if ($p) {
        Write-Host ("  - killing PID " + $pid + " (" + $p.ProcessName + ")") -ForegroundColor DarkYellow
        Stop-Process -Id $pid -Force
      }
    } catch {}
  }
}

function Stop-ExeByName($name) {
  Get-Process -Name $name -ErrorAction SilentlyContinue | ForEach-Object {
    Write-Host ("  - killing " + $_.ProcessName + " (PID " + $_.Id + ")") -ForegroundColor DarkYellow
    Stop-Process -Id $_.Id -Force
  }
}

Write-Step "=== desktop-follow one-click start ==="
Stop-Occupying $VitePort
Stop-ExeByName $ExeName

Write-Step ("Starting Vite on http://localhost:" + $VitePort + " (hidden window)")
$ViteLog    = Join-Path $LogDir 'vite.log'
$ViteErrLog = Join-Path $LogDir 'vite.err.log'
Start-Process -FilePath "cmd.exe" -ArgumentList @('/c','pnpm','dev') `
  -RedirectStandardOutput $ViteLog `
  -RedirectStandardError $ViteErrLog `
  -WorkingDirectory $ProjectRoot `
  -WindowStyle Hidden
Write-Host ("  log: " + $ViteLog) -ForegroundColor DarkGray

Write-Step ("Waiting for " + $VitePort + " to answer 200 ...")
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
  Start-Sleep -Seconds 1
  if (Test-PortOpen -Port $VitePort) { $ready = $true; break }
}
if (-not $ready) {
  Write-Host ("Vite did not start in 30s — see " + $ViteLog) -ForegroundColor Red
  exit 1
}
Write-Host "Vite ready" -ForegroundColor Green

if (-not (Test-Path $ExePath)) {
  Write-Host ("Missing: " + $ExePath) -ForegroundColor Red
  Write-Host "Hint: cd src-tauri && cargo build (first build takes 5-10 min)" -ForegroundColor Yellow
  exit 1
}

Write-Step ("Starting Tauri (hidden): " + $ExePath)
$ExeLog    = Join-Path $LogDir 'tauri.log'
$ExeErrLog = Join-Path $LogDir 'tauri.err.log'
Start-Process -FilePath $ExePath `
  -RedirectStandardOutput $ExeLog `
  -RedirectStandardError $ExeErrLog `
  -WindowStyle Hidden
Write-Host ("  log: " + $ExeLog) -ForegroundColor DarkGray
Start-Sleep -Seconds 2

Get-Process -Name $ExeName -ErrorAction SilentlyContinue | Format-Table Id,ProcessName -AutoSize | Out-String | Write-Host

Write-Host ""
Write-Host "[OK] started. Press Ctrl+C in this window to stop the watcher." -ForegroundColor Green
Write-Host "Children stay alive even after this script exits." -ForegroundColor DarkGray
Write-Host "Use scripts\dev-stop.ps1 to stop them manually." -ForegroundColor DarkGray

try {
  while ($true) { Start-Sleep -Seconds 60 }
}
finally {
  Write-Host "`nWatcher ended. Children still running (use dev-stop.ps1 to kill)." -ForegroundColor Cyan
}