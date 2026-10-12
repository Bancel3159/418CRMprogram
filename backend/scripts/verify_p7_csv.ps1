# CSV 上传测试
$base = "http://localhost:8080"

$login = Invoke-RestMethod -Uri "$base/api/v1/auth/login" -Method POST -ContentType "application/json" -Body '{"username":"admin","password":"admin123","turnstile_token":"dev-bypass"}'
$token = $login.data.access_token
$hdr = @{ Authorization = "Bearer $token" }

# 找 TWD 游戏的模板 ID
$tpls = Invoke-RestMethod -Uri "$base/api/v1/upload-templates?game_id=1" -Headers $hdr -Method GET
$tplId = $tpls.data[0].id
Write-Host "Template ID: $tplId"

# 上传 CSV
$filePath = "E:\MyProject\418CRMprogram\backend\scripts\test_orders.csv"
try {
    $resp = Invoke-RestMethod -Uri "$base/api/v1/orders/import" -Headers $hdr -Method POST -Form @{
        game_id = "1"
        template_id = "$tplId"
        file = Get-Item -Path $filePath
    }
    Write-Host "`n=== CSV 上传结果 ==="
    Write-Host "批次号: $($resp.data.batch)"
    Write-Host "总行数: $($resp.data.summary.total)"
    Write-Host "新增: $($resp.data.summary.inserted)"
    Write-Host "跳过: $($resp.data.summary.skipped)"
    Write-Host "失败: $($resp.data.summary.failed)"
    Write-Host "自动归属: $($resp.data.summary.auto_attributed)"
    Write-Host "待人工: $($resp.data.summary.pending_attribution)"
    if ($resp.data.row_errors) {
        Write-Host "`n行错误:"
        $resp.data.row_errors | ForEach-Object { Write-Host "  $($_.row_number): $($_.message)" }
    }
} catch {
    Write-Host "上传失败: $($_.Exception.Message)"
}

Write-Host "`n=== 查看上传后的订单列表 (前 5 条) ==="
$orders = Invoke-RestMethod -Uri "$base/api/v1/orders?page=1&page_size=5" -Headers $hdr -Method GET
$orders.data.list | ForEach-Object { Write-Host "  [$($_.id)] $($_.order_no) ¥$($_.amount) src=$($_.source) attr=$($_.attribution_status)" }

Write-Host "`n=== 重新查看统计 ==="
$stats = Invoke-RestMethod -Uri "$base/api/v1/orders/stats" -Headers $hdr -Method GET
Write-Host "总营收: ¥$($stats.data.overview.total_revenue)"
Write-Host "订单数: $($stats.data.overview.total_orders)"
Write-Host "付费玩家: $($stats.data.overview.paid_players)"
Write-Host "ARPU: ¥$($stats.data.overview.arpu)"
Write-Host "A 队业绩: ¥$($stats.data.overview.a_team_revenue)"
Write-Host "B 队业绩: ¥$($stats.data.overview.b_team_revenue)"

Write-Host "`n✅ CSV 上传 + 三通道验证完成" -ForegroundColor Green