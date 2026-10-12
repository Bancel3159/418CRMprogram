@echo off
setlocal

REM 登录
curl -s -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"password\":\"admin123\",\"turnstile_token\":\"dev-bypass\"}" > login.json

REM 提取 token (使用 powershell 处理 JSON)
for /f "delims=" %%i in ('powershell -NoProfile -Command "(Get-Content login.json | ConvertFrom-Json).data.access_token"') do set TOKEN=%%i

echo TOKEN length:
echo %TOKEN% | findstr /R "^" > nul && powershell -NoProfile -Command "Write-Host ('Length: ' + '%TOKEN%'.Length)"

REM 查找模板
curl -s "http://localhost:8080/api/v1/upload-templates?game_id=1" -H "Authorization: Bearer %TOKEN%" > tpls.json
for /f "delims=" %%i in ('powershell -NoProfile -Command "(Get-Content tpls.json | ConvertFrom-Json).data[0].id"') do set TPL_ID=%%i
echo Template ID: %TPL_ID%

REM 上传 CSV
echo.
echo === CSV 上传 ===
curl -s -X POST http://localhost:8080/api/v1/orders/import -H "Authorization: Bearer %TOKEN%" -F "game_id=1" -F "template_id=%TPL_ID%" -F "file=@E:\MyProject\418CRMprogram\backend\scripts\test_orders.csv"

REM 查看统计
echo.
echo === 更新后的统计 ===
curl -s "http://localhost:8080/api/v1/orders/stats" -H "Authorization: Bearer %TOKEN%"

echo.
del login.json tpls.json 2>nul