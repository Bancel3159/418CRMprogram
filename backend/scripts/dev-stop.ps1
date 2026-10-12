<#
.SYNOPSIS
  Stop the CRM backend server started by dev-start.ps1.
.NOTES
  - Kills any process listening on port 8080
  - 不会停 Docker 里的 PostgreSQL / Redis
#>
$ErrorActionPreference = 'Stop'

$ServerPort = 8080
$killed = $false

Write-Host ("Stopping process on port " + $ServerPort + " ...") -ForegroundColor Cyan
$conns = Get-NetTCPConnection -LocalPort $ServerPort -State Listen -ErrorAction SilentlyContinue
foreach ($c in $conns) {
  $procId = $c.OwningProcess
  try {
    $p = Get-Process -Id $procId -ErrorAction SilentlyContinue
    if ($p) {
      Write-Host ("  - killing PID " + $procId + " (" + $p.ProcessName + ")") -ForegroundColor DarkYellow
      Stop-Process -Id $procId -Force
      $killed = $true
    }
  } catch {}
}

if (-not $killed) {
  Write-Host "No process found on port $ServerPort." -ForegroundColor DarkGray
} else {
  Write-Host "[OK] backend stopped." -ForegroundColor Green
}