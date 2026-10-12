#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""P8 操作日志验证"""
import sys, json, time
sys.stdout = open(sys.stdout.fileno(), mode='w', encoding='utf-8', buffering=1)
import urllib.request, urllib.error, hmac, hashlib

BASE = "http://localhost:8080"

def req(method, url, data=None, headers=None):
    if isinstance(data, (dict, list)):
        body = json.dumps(data).encode()
    elif isinstance(data, bytes):
        body = data
    else:
        body = b""
    h = headers.copy() if headers else {}
    if body and "Content-Type" not in h:
        h["Content-Type"] = "application/json"
    r = urllib.request.Request(url, data=body if body else None, method=method, headers=h)
    try:
        with urllib.request.urlopen(r, timeout=60) as resp:
            return resp.status, json.loads(resp.read().decode() or "null")
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode() or "{}")

# 登录
code, r = req("POST", f"{BASE}/api/v1/auth/login",
              {"username": "admin", "password": "admin123", "turnstile_token": "dev-bypass"})
TOKEN = r['data']['access_token']
H = {"Authorization": f"Bearer {TOKEN}"}
print(f"[1] 登录: HTTP {code}")

# 触发各种动作
print("\n[2] 触发各种审计动作...")

# 2.1 登录失败(用错密码)
code, _ = req("POST", f"{BASE}/api/v1/auth/login",
              {"username": "admin", "password": "WRONG", "turnstile_token": "dev-bypass"})
print(f"  登录失败: HTTP {code}")

# 2.2 重置某个用户密码(找 user id=2)
code, r = req("POST", f"{BASE}/api/v1/users/2/reset-password", data={"new_password": "test123"}, headers=H)
print(f"  重置密码: HTTP {code}")

# 2.3 Webhook 入库
secret = "demo-secret-rotate-me"
wh_body = json.dumps({
    "orders": [
        {
            "order_no": f"AUDIT-WH-{int(time.time())}",
            "player_id": "VITE_PLAYER_001",
            "game_id": 1,
            "amount": 30.0,
            "currency": "CNY",
            "status": "paid",
            "order_time": "2026-09-25 22:00:00"
        }
    ]
})
sig = hmac.new(secret.encode(), wh_body.encode(), hashlib.sha256).hexdigest()
code, _ = req("POST", f"{BASE}/api/v1/orders/webhook/demo-webhook", data=wh_body.encode(),
              headers={"X-Signature": f"sha256={sig}", "Content-Type": "application/json"})
print(f"  Webhook 入库: HTTP {code}")

# 2.4 CSV 导入
boundary = "----TestBoundary"
csv = "订单号,玩家ID,充值金额,支付状态,下单时间\nAUDIT-CSV-001,VITE_PLAYER_001,99.00,已支付,2026-09-25 23:00:00\n".encode("utf-8")
body = b""
for k, v in [("game_id", "1"), ("template_id", "1")]:
    body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"{k}\"\r\n\r\n{v}\r\n".encode()
body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.csv\"\r\nContent-Type: text/csv\r\n\r\n".encode()
body += csv
body += f"\r\n--{boundary}--\r\n".encode()
code, _ = req("POST", f"{BASE}/api/v1/orders/import", data=body,
              headers={**H, "Content-Type": f"multipart/form-data; boundary={boundary}"})
print(f"  CSV 导入: HTTP {code}")

# 等待异步审计写入
time.sleep(2)

# 3. 查询审计日志
print("\n[3] 查询审计日志(最近 10 条)...")
code, r = req("GET", f"{BASE}/api/v1/audit/logs?page=1&page_size=10", headers=H)
print(f"  HTTP {code} | total: {r['data']['total']}")
for log in r['data']['list'][:10]:
    print(f"  [{log['id']}] {log['created_at'][:19]} {log['username'] or '-':12} {log['action']:20} → {log['target_type']} #{log['target_id'] or '-'} [{log['status']}]")

# 4. 审计统计
print("\n[4] 审计统计(近 7 天)...")
code, r = req("GET", f"{BASE}/api/v1/audit/stats", headers=H)
print(f"  HTTP {code}")
print("  按动作分布:")
for row in r['data']['by_action'][:8]:
    print(f"    {row['action']:20} {row['count']}")
print("  按用户 TOP 10:")
for row in r['data']['by_user'][:5]:
    print(f"    {row['username']:12} {row['count']}")

# 5. 过滤测试
print("\n[5] 过滤测试:只看待失败的...")
code, r = req("GET", f"{BASE}/api/v1/audit/logs?status=failure", headers=H)
print(f"  HTTP {code} | 失败条数: {r['data']['total']}")
for log in r['data']['list'][:3]:
    print(f"    [{log['id']}] {log['action']:20} {log['username']:12} - {log['error_message']}")

print("\n" + "=" * 50)
print("✅ P8 操作日志验证完成")
print("=" * 50)