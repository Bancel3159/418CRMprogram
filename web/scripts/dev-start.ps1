<#
.SYNOPSIS
  One-click dev starter for web admin (CRM 后台管理 web 端).
.NOTES
  - 仅起 Vite(port 5273),没有 Tauri exe
  - Vite 启动后保持常驻;关闭本脚本不影响 Vite
  - 需要 backend (:8080) 已先启动 —— Vite 的 /api、/uploads、/ws 代理都指向 :8080
  - Flow:
    1. Kill leftovers on port 5273
    2. Background-start pnpm dev (logs in .dev-logs/vite.log)
    3. Poll localhost:5273 with TCP until answer(超时 30s)
  Press Ctrl+C in this window to stop the watcher (children stay alive).
#>
$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

$VitePort = 5273
$LogDir   = Join-Path $ProjectRoot '.dev-logs'
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
    $procId = $c.OwningProcess   # 不要用 $pid(PowerShell 保留变量,只读)
    try {
      $p = Get-Process -Id $procId -ErrorAction SilentlyContinue
      if ($p) {
        Write-Host ("  - killing PID " + $procId + " (" + $p.ProcessName + ")") -ForegroundColor DarkYellow
        Stop-Process -Id $procId -Force
      }
    } catch {}
  }
}

Write-Step "=== web admin one-click start ==="
Stop-Occupying $VitePort

Write-Step ("Starting Vite on http://localhost:" + $VitePort + " (hidden window)")
$ViteLog    = Join-Path $LogDir 'vite.log'
$ViteErrLog = Join-Path $LogDir 'vite.err.log'
Start-Process -FilePath "cmd.exe" -ArgumentList @('/c','pnpm','dev') `
  -RedirectStandardOutput $ViteLog `
  -RedirectStandardError $ViteErrLog `
  -WorkingDirectory $ProjectRoot `
  -WindowStyle Hidden
Write-Host ("  log: " + $ViteLog) -ForegroundColor DarkGray

Write-Step ("Waiting for " + $VitePort + " to answer ...")
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

Write-Host ""
Write-Host "[OK] web admin started at http://localhost:$VitePort" -ForegroundColor Green
Write-Host "Default admin account: admin / admin123" -ForegroundColor DarkGray
Write-Host "Children stay alive even after this script exits." -ForegroundColor DarkGray
Write-Host "Use scripts\dev-stop.ps1 to stop Vite." -ForegroundColor DarkGray

# Keep this terminal alive until Ctrl+C; do NOT auto-clean children on exit.
try {
  while ($true) { Start-Sleep -Seconds 60 }
}
finally {
  Write-Host "`nWatcher ended. Vite still running (use dev-stop.ps1 to kill)." -ForegroundColor Cyan
}