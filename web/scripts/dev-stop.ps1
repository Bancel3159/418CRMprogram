<#
.SYNOPSIS
  Stop the web admin dev process (Vite on 5173).
#>
$ErrorActionPreference = 'SilentlyContinue'

Write-Host "=== Stopping web admin ===" -ForegroundColor Cyan

# Kill anything listening on 5173 (Vite)
$conns = Get-NetTCPConnection -LocalPort 5173 -State Listen -ErrorAction SilentlyContinue
foreach ($c in $conns) {
  $p = Get-Process -Id $c.OwningProcess -ErrorAction SilentlyContinue
  if ($p) {
    Write-Host "  - Killing $($p.ProcessName) (PID $($p.Id))"
    Stop-Process -Id $p.Id -Force
  }
}

Write-Host "[OK] cleaned" -ForegroundColor Green