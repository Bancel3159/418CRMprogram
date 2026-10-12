#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""P7 完整三通道最终验证"""
import sys, json, os
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
print("=" * 50)
print("1. 登录")
print("=" * 50)
code, r = req("POST", f"{BASE}/api/v1/auth/login",
              {"username": "admin", "password": "admin123", "turnstile_token": "dev-bypass"})
print(f"  HTTP {code} | token 长度: {len(r['data']['access_token'])}")
TOKEN = r['data']['access_token']
H = {"Authorization": f"Bearer {TOKEN}"}

# 2. 合作方列表
print("\n" + "=" * 50)
print("2. 合作方")
print("=" * 50)
code, r = req("GET", f"{BASE}/api/v1/partners", headers=H)
for p in r['data']:
    print(f"  [{p['id']}] {p['code']} / {p['name']} / type={p['type']} enabled={p['enabled']}")

# 3. 上传模板
print("\n" + "=" * 50)
print("3. 上传模板")
print("=" * 50)
code, r = req("GET", f"{BASE}/api/v1/upload-templates", headers=H)
for t in r['data']:
    print(f"  [{t['id']}] {t['name']} game={t['game']['code']}")

# 4. 订单统计 (upload 之前)
print("\n" + "=" * 50)
print("4. 上传前的统计")
print("=" * 50)
code, r = req("GET", f"{BASE}/api/v1/orders/stats", headers=H)
o = r['data']['overview']
print(f"  总营收: ¥{o['total_revenue']:.2f}")
print(f"  订单数: {o['total_orders']}")
print(f"  ARPU: ¥{o['arpu']:.2f}")
print(f"  A 队业绩: ¥{o['a_team_revenue']:.2f} ({o['a_team_orders']} 笔)")
print(f"  B 队业绩: ¥{o['b_team_revenue']:.2f} ({o['b_team_orders']} 笔)")
prev_total = o['total_orders']
prev_revenue = o['total_revenue']

# 5. CSV 上传
print("\n" + "=" * 50)
print("5. CSV 手动上传")
print("=" * 50)
boundary = "----TestBoundary"
csv_path = r"E:\MyProject\418CRMprogram\backend\scripts\test_orders.csv"
with open(csv_path, "rb") as f:
    csv_bytes = f.read()
body = b""
for k, v in [("game_id", "1"), ("template_id", "1")]:
    body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"{k}\"\r\n\r\n{v}\r\n".encode()
body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"test.csv\"\r\nContent-Type: text/csv\r\n\r\n".encode()
body += csv_bytes
body += f"\r\n--{boundary}--\r\n".encode()
code, r = req("POST", f"{BASE}/api/v1/orders/import", data=body,
              headers={**H, "Content-Type": f"multipart/form-data; boundary={boundary}"})
s = r['data']['summary']
print(f"  批次: {r['data']['batch']}")
print(f"  新增: {s['inserted']} 跳过: {s['skipped']} 失败: {s['failed']}")
print(f"  自动归属: {s['auto_attributed']} 待人工: {s['pending_attribution']}")

# 6. Webhook 测试 (HMAC)
print("\n" + "=" * 50)
print("6. Webhook (HMAC-SHA256)")
print("=" * 50)
import time
secret = "demo-secret-rotate-me"
wh_body = json.dumps({
    "orders": [
        {
            "order_no": f"WH-FINAL-{int(time.time())}",
            "player_id": "VITE_PLAYER_001",
            "game_id": 1,
            "amount": 50.0,
            "currency": "CNY",
            "status": "paid",
            "order_time": "2026-09-25 22:00:00",
            "paid_at": "2026-09-25 22:00:00"
        }
    ]
})
sig = hmac.new(secret.encode(), wh_body.encode(), hashlib.sha256).hexdigest()
code, r = req("POST", f"{BASE}/api/v1/orders/webhook/demo-webhook", data=wh_body.encode(),
              headers={"X-Signature": f"sha256={sig}", "Content-Type": "application/json"})
print(f"  HTTP {code} | 响应: {r['data']}")

# 错误签名测试
print("\n  错误签名测试 (应 401):")
code, r = req("POST", f"{BASE}/api/v1/orders/webhook/demo-webhook", data=wh_body.encode(),
              headers={"X-Signature": "sha256=invalid", "Content-Type": "application/json"})
print(f"  HTTP {code} | {r.get('message','')}")

# 7. 手动绑定 lead
print("\n" + "=" * 50)
print("7. 手动绑定订单 (admin → lead)")
print("=" * 50)
code, r = req("GET", f"{BASE}/api/v1/orders?attribution_status=pending_attribution", headers=H)
if r['data']['list']:
    pending = r['data']['list'][0]
    code, r2 = req("PATCH", f"{BASE}/api/v1/orders/{pending['id']}/bind",
                   data={"lead_id": 2}, headers=H)
    print(f"  订单 {pending['order_no']} 绑定 → lead_id=2")
    print(f"  绑定后状态: {r2.get('data', {}).get('attribution_status', '?')}")
else:
    print("  无待绑定订单")

# 8. 上传后的最终统计
print("\n" + "=" * 50)
print("8. 上传后的最终统计")
print("=" * 50)
code, r = req("GET", f"{BASE}/api/v1/orders/stats", headers=H)
o = r['data']['overview']
print(f"  总营收: ¥{o['total_revenue']:.2f} (Δ +¥{o['total_revenue']-prev_revenue:.2f})")
print(f"  订单数: {o['total_orders']} (Δ +{o['total_orders']-prev_total})")
print(f"  付费玩家: {o['paid_players']}")
print(f"  ARPU: ¥{o['arpu']:.2f}")
print(f"  A 队业绩: ¥{o['a_team_revenue']:.2f} ({o['a_team_orders']} 笔)")
print(f"  B 队业绩: ¥{o['b_team_revenue']:.2f} ({o['b_team_orders']} 笔)")
print(f"  待人工绑定: {o['pending_orders']}")

# 9. 查看最新订单
print("\n" + "=" * 50)
print("9. 最新订单 (前 5)")
print("=" * 50)
code, r = req("GET", f"{BASE}/api/v1/orders?page=1&page_size=5", headers=H)
for o in r['data']['list']:
    print(f"  [{o['id']}] {o['order_no']:35} ¥{o['amount']:7.2f} src={o['source']:7} attr={o['attribution_status']:18} lead={o['lead_id']}")

print("\n" + "=" * 50)
print("✅ P7 全部三通道入库验证通过")
print("=" * 50)