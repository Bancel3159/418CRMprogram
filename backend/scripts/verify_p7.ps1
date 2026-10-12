# P7 端到端验证脚本

$base = "http://localhost:8080"
$ErrorActionPreference = "Continue"

Write-Host "=== 1. 登录 ===" -ForegroundColor Cyan
$login = Invoke-RestMethod -Uri "$base/api/v1/auth/login" -Method POST -ContentType "application/json" -Body '{"username":"admin","password":"admin123","turnstile_token":"dev-bypass"}'
$token = $login.data.access_token
$hdr = @{ Authorization = "Bearer $token" }
Write-Host "Token: $($token.Substring(0,30))..."

Write-Host "`n=== 2. 合作方列表 ===" -ForegroundColor Cyan
$partners = Invoke-RestMethod -Uri "$base/api/v1/partners" -Headers $hdr -Method GET
$partners.data | ForEach-Object { Write-Host "  [$($_.id)] $($_.code) / $($_.name) / type=$($_.type)" }

Write-Host "`n=== 3. 上传模板列表 ===" -ForegroundColor Cyan
$tpls = Invoke-RestMethod -Uri "$base/api/v1/upload-templates" -Headers $hdr -Method GET
$tpls.data | ForEach-Object { Write-Host "  [$($_.id)] $($_.name) game_id=$($_.game_id)" }

Write-Host "`n=== 4. 订单列表 ===" -ForegroundColor Cyan
$orders = Invoke-RestMethod -Uri "$base/api/v1/orders" -Headers $hdr -Method GET
Write-Host "Total: $($orders.data.total)"
$orders.data.list | ForEach-Object { Write-Host "  [$($_.id)] $($_.order_no) ¥$($_.amount) src=$($_.source) attr=$($_.attribution_status) lead=$($_.lead_id)" }

Write-Host "`n=== 5. 订单统计 ===" -ForegroundColor Cyan
$stats = Invoke-RestMethod -Uri "$base/api/v1/orders/stats" -Headers $hdr -Method GET
Write-Host "总营收: ¥$($stats.data.overview.total_revenue)"
Write-Host "总订单数: $($stats.data.overview.total_orders)"
Write-Host "付费玩家: $($stats.data.overview.paid_players)"
Write-Host "ARPU: ¥$($stats.data.overview.arpu)"
Write-Host "待人工: $($stats.data.overview.pending_orders)"
Write-Host "A 团队业绩: ¥$($stats.data.overview.a_team_revenue) ($($stats.data.overview.a_team_orders) 单)"
Write-Host "B 团队业绩: ¥$($stats.data.overview.b_team_revenue) ($($stats.data.overview.b_team_orders) 单)"

Write-Host "`n=== 6. 仅看待绑定订单 ===" -ForegroundColor Cyan
$pending = Invoke-RestMethod -Uri "$base/api/v1/orders?unbound=true" -Headers $hdr -Method GET
$pending.data.list | ForEach-Object { Write-Host "  [$($_.id)] $($_.order_no) player=$($_.player_id) lead=$($_.lead_id)" }

Write-Host "`n=== 7. Webhook 测试 (HMAC 验签) ===" -ForegroundColor Cyan
$body = '{"orders":[{"order_no":"WH-TEST-001","player_id":"VITE_PLAYER_001","game_id":1,"amount":99.50,"currency":"CNY","status":"paid","order_time":"2026-09-25 14:00:00","paid_at":"2026-09-25 14:00:00"}]}'
$secret = "demo-secret-rotate-me"
$sig = -join ([System.Security.Cryptography.HMACSHA256]::new([System.Text.Encoding]::UTF8.GetBytes($secret)).ComputeHash([System.Text.Encoding]::UTF8.GetBytes($body)) | ForEach-Object { $_.ToString("x2") })

$whResp = Invoke-RestMethod -Uri "$base/api/v1/orders/webhook/demo-webhook" -Method POST -ContentType "application/json" -Headers @{ "X-Signature" = "sha256=$sig" } -Body $body
Write-Host "Webhook 响应: $($whResp | ConvertTo-Json -Depth 3 -Compress)"

Write-Host "`n=== 8. Webhook 错误签名测试 (应被拒) ===" -ForegroundColor Cyan
try {
    $bad = Invoke-RestMethod -Uri "$base/api/v1/orders/webhook/demo-webhook" -Method POST -ContentType "application/json" -Headers @{ "X-Signature" = "sha256=invalid" } -Body $body
    Write-Host "未拒绝? $($bad.secret)" -ForegroundColor Red
} catch {
    Write-Host "正确拒绝: $($_.Exception.Message)" -ForegroundColor Green
}

Write-Host "`n=== 9. 手动绑定订单 ===" -ForegroundColor Cyan
# 找到待绑定的订单
$pendingOrder = $pending.data.list[0]
if ($pendingOrder) {
    # 找一个匹配的 lead (这里随便用 leads 数据)
    $leadId = $orders.data.list[1].lead_id  # 借用已归属订单的 lead
    Write-Host "绑定订单 [$($pendingOrder.id)] 到 lead [$leadId]"
    $bindResp = Invoke-RestMethod -Uri "$base/api/v1/orders/$($pendingOrder.id)/bind" -Headers $hdr -Method PATCH -ContentType "application/json" -Body "{`"lead_id`":$leadId}"
    Write-Host "绑定后 attribution_status = $($bindResp.data.attribution_status), lead_id = $($bindResp.data.lead_id)"
}

Write-Host "`n=== 10. 合作方手动 API 拉取 (会失败:外部 URL) ===" -ForegroundColor Cyan
try {
    $pullResp = Invoke-RestMethod -Uri "$base/api/v1/partners/2/pull" -Headers $hdr -Method POST
    Write-Host "拉取响应: $($pullResp | ConvertTo-Json -Depth 3 -Compress)"
} catch {
    Write-Host "预期失败 (演示地址): $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host "`n=== 11. 重新查看统计 (反映 webhook 新订单) ===" -ForegroundColor Cyan
$stats2 = Invoke-RestMethod -Uri "$base/api/v1/orders/stats" -Headers $hdr -Method GET
Write-Host "总营收(已含 webhook): ¥$($stats2.data.overview.total_revenue)"
Write-Host "总订单数: $($stats2.data.overview.total_orders)"

Write-Host "`n✅ 全部验证完成" -ForegroundColor Green