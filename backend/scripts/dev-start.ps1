<#
.SYNOPSIS
  One-click dev starter for the CRM Go backend (Gin + GORM).
.NOTES
  - 后端 Go 后端目前不在 docker 里 —— docker-compose 只起 PostgreSQL + Redis
  - 使用预编译的 bin/server.exe;若代码改了需先 `go build -o bin/server.exe ./cmd/server` 再启
  - 数据库/Redis 默认连 localhost:5432 / localhost:6379,沿用 backend/.env
  - Flow:
    1. Kill leftovers on port 8080
    2. Background-start bin/server.exe (logs in .dev-logs/server.log)
    3. Poll localhost:8080 with TCP until answer (timeout 20s)
  Press Ctrl+C in this window to stop the watcher (children stay alive).
#>
$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

$ServerPort = 8080
$ExePath    = Join-Path $ProjectRoot 'bin\server.exe'
$LogDir     = Join-Path $ProjectRoot '.dev-logs'
if (-not (Test-Path $LogDir)) { New-Item -ItemType Directory -Path $LogDir | Out-Null }

# 强制 PowerShell 控制台用 UTF-8 读日志,避免中文乱码
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

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

Write-Step "=== CRM backend one-click start ==="

if (-not (Test-Path $ExePath)) {
  Write-Host ("未找到 " + $ExePath + " —— 请先编译:") -ForegroundColor Red
  Write-Host "  cd backend && go build -o bin/server.exe ./cmd/server" -ForegroundColor Yellow
  exit 1
}

Stop-Occupying $ServerPort

Write-Step ("Starting " + $ExePath + " on http://localhost:" + $ServerPort + " (hidden window)")
$ServerLog    = Join-Path $LogDir 'server.log'
$ServerErrLog = Join-Path $LogDir 'server.err.log'
Start-Process -FilePath $ExePath `
  -RedirectStandardOutput $ServerLog `
  -RedirectStandardError $ServerErrLog `
  -WorkingDirectory $ProjectRoot `
  -WindowStyle Hidden
Write-Host ("  log:    " + $ServerLog) -ForegroundColor DarkGray
Write-Host ("  errlog: " + $ServerErrLog) -ForegroundColor DarkGray

Write-Step ("Waiting for " + $ServerPort + " to answer ...")
$ready = $false
for ($i = 0; $i -lt 20; $i++) {
  Start-Sleep -Seconds 1
  if (Test-PortOpen -Port $ServerPort) { $ready = $true; break }
}
if (-not $ready) {
  Write-Host ("Server did not start in 20s — see " + $ServerLog) -ForegroundColor Red
  exit 1
}
Write-Host "Server ready" -ForegroundColor Green

Write-Host ""
Write-Host "[OK] backend started at http://localhost:$ServerPort" -ForegroundColor Green
Write-Host "DB:    PostgreSQL @ localhost:5432 (via Docker)" -ForegroundColor DarkGray
Write-Host "Cache: Redis      @ localhost:6379 (via Docker)" -ForegroundColor DarkGray
Write-Host ""
Write-Host "server.exe is detached from this shell — it will keep running after the script exits." -ForegroundColor DarkGray
Write-Host "Verify:  Invoke-WebRequest http://localhost:$ServerPort/health" -ForegroundColor DarkGray
Write-Host "Stop:    scripts\dev-stop.ps1   (kills whatever owns :$ServerPort)" -ForegroundColor DarkGray

# 脚本立即退出,server.exe 由 Start-Process 创建为孤儿进程继续运行
exit 0