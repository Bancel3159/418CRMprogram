<#
.SYNOPSIS
  Stop the dev processes for desktop-follow (Vite on 5175 + crm-desktop-follow.exe).
#>
$ErrorActionPreference = 'SilentlyContinue'

Write-Host "=== Stopping desktop-follow ===" -ForegroundColor Cyan

Get-Process -Name 'crm-desktop-follow' -ErrorAction SilentlyContinue | ForEach-Object {
  Write-Host "  - Killing crm-desktop-follow (PID $($_.Id))"
  Stop-Process -Id $_.Id -Force
}

$conns = Get-NetTCPConnection -LocalPort 5175 -State Listen -ErrorAction SilentlyContinue
foreach ($c in $conns) {
  $p = Get-Process -Id $c.OwningProcess -ErrorAction SilentlyContinue
  if ($p) {
    Write-Host "  - Killing $($p.ProcessName) (PID $($p.Id))"
    Stop-Process -Id $p.Id -Force
  }
}

Write-Host "[OK] cleaned" -ForegroundColor Green