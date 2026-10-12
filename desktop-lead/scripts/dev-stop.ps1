<#
.SYNOPSIS
  Stop the dev processes for desktop-lead (Vite on 5174 + crm-desktop-lead.exe).
#>
$ErrorActionPreference = 'SilentlyContinue'

Write-Host "=== Stopping desktop-lead ===" -ForegroundColor Cyan

# Kill the Tauri window exe
Get-Process -Name 'crm-desktop-lead' -ErrorAction SilentlyContinue | ForEach-Object {
  Write-Host "  - Killing crm-desktop-lead (PID $($_.Id))"
  Stop-Process -Id $_.Id -Force
}

# Kill anything listening on 5174 (Vite)
$conns = Get-NetTCPConnection -LocalPort 5174 -State Listen -ErrorAction SilentlyContinue
foreach ($c in $conns) {
  $p = Get-Process -Id $c.OwningProcess -ErrorAction SilentlyContinue
  if ($p) {
    Write-Host "  - Killing $($p.ProcessName) (PID $($p.Id))"
    Stop-Process -Id $p.Id -Force
  }
}

Write-Host "[OK] cleaned" -ForegroundColor Green