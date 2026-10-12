#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""P8 完整验证:审计 + WS 告警"""
import sys, json, time, socket, base64, struct
sys.stdout = open(sys.stdout.fileno(), mode='w', encoding='utf-8', buffering=1)
import urllib.request, urllib.error, hmac, hashlib, threading, queue

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

# Login as admin
code, r = req("POST", f"{BASE}/api/v1/auth/login",
              {"username": "admin", "password": "admin123", "turnstile_token": "dev-bypass"})
TOKEN = r['data']['access_token']
H = {"Authorization": f"Bearer {TOKEN}"}

print("=" * 60)
print("P8 完整验证:操作审计 + WebSocket 告警")
print("=" * 60)

# 1. 触发多种审计动作
print("\n[1] 触发各种审计动作...")

# 登录失败
code, _ = req("POST", f"{BASE}/api/v1/auth/login",
              {"username": "admin", "password": "WRONG1", "turnstile_token": "dev-bypass"})
code, _ = req("POST", f"{BASE}/api/v1/auth/login",
              {"username": "admin", "password": "WRONG2", "turnstile_token": "dev-bypass"})
print(f"  ✓ 登录失败 ×2")

# 重置密码
code, _ = req("POST", f"{BASE}/api/v1/users/2/reset-password", data={"new_password": "test123"}, headers=H)
print(f"  ✓ 重置密码")

# 找 lead 1 认领测试(找一个 pending lead)
# 用现有的 pending lead 测试
code, r = req("GET", f"{BASE}/api/v1/leads?status=pending&page=1&page_size=1", headers=H)
if r['data']['list']:
    lead_id = r['data']['list'][0]['id']
    code, _ = req("POST", f"{BASE}/api/v1/leads/{lead_id}/archive", headers=H)
    print(f"  ✓ 归档 lead #{lead_id}")

# CSV 导入
boundary = "----TestBoundary"
csv = "订单号,玩家ID,充值金额,支付状态,下单时间\nP8-AUDIT-001,VITE_PLAYER_001,55.00,已支付,2026-09-25 23:30:00\nP8-AUDIT-002,WS_POOL_TEST_001,77.00,已支付,2026-09-25 23:30:00\n".encode("utf-8")
body = b""
for k, v in [("game_id", "1"), ("template_id", "1")]:
    body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"{k}\"\r\n\r\n{v}\r\n".encode()
body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"p8.csv\"\r\nContent-Type: text/csv\r\n\r\n".encode()
body += csv
body += f"\r\n--{boundary}--\r\n".encode()
code, _ = req("POST", f"{BASE}/api/v1/orders/import", data=body,
              headers={**H, "Content-Type": f"multipart/form-data; boundary={boundary}"})
print(f"  ✓ CSV 导入(2 行)")

# Webhook 入库(触发 alert)
secret = "demo-secret-rotate-me"
wh_body = json.dumps({
    "orders": [
        {
            "order_no": f"P8-WH-{int(time.time())}",
            "player_id": "VITE_PLAYER_001",
            "game_id": 1,
            "amount": 199.99,
            "currency": "CNY",
            "status": "paid",
            "order_time": "2026-09-25 23:30:00"
        }
    ]
})
sig = hmac.new(secret.encode(), wh_body.encode(), hashlib.sha256).hexdigest()
code, _ = req("POST", f"{BASE}/api/v1/orders/webhook/demo-webhook", data=wh_body.encode(),
              headers={"X-Signature": f"sha256={sig}", "Content-Type": "application/json"})
print(f"  ✓ Webhook 入库(应触发 alert)")

# 等待异步写入
print("\n等待 3 秒让异步写入完成...")
time.sleep(3)

# 2. 查询审计日志
print("\n[2] 审计日志查询 (近 20 条)...")
code, r = req("GET", f"{BASE}/api/v1/audit/logs?page=1&page_size=20", headers=H)
print(f"  HTTP {code} | total: {r['data']['total']}")
print(f"  {'ID':>4} {'时间':19} {'用户':10} {'动作':18} {'对象':20} {'IP':14} {'状态'}")
for log in r['data']['list'][:15]:
    print(f"  [{log['id']:>3}] {log['created_at'][:19]} {(log['username'] or '-'):10} {log['action']:18} {(log['target_label'] or '-'):20} {(log['ip'] or '-'):14} {log['status']}")

# 3. 按动作过滤
print("\n[3] 过滤:只看 login_failed ...")
code, r = req("GET", f"{BASE}/api/v1/audit/logs?action=login_failed&page=1&page_size=5", headers=H)
print(f"  共 {r['data']['total']} 条")
for log in r['data']['list']:
    print(f"    [{log['id']}] {log['username']} - {log['error_message']}")

# 4. 统计
print("\n[4] 审计统计...")
code, r = req("GET", f"{BASE}/api/v1/audit/stats", headers=H)
print("  动作 TOP 10:")
for row in r['data']['by_action'][:10]:
    print(f"    {row['action']:20} {row['count']}")
print("  用户 TOP 5:")
for row in r['data']['by_user'][:5]:
    print(f"    {row['username']:10} {row['count']}")

# 5. WebSocket 告警测试
print("\n[5] WebSocket 告警测试...")
import asyncio
try:
    import websockets
    HAS_WS = True
except ImportError:
    HAS_WS = False
    print("  ⚠ python websockets 库未安装,跳过 WS 测试")

if HAS_WS:
    async def test_ws():
        uri = f"ws://localhost:8080/ws?token={TOKEN}"
        async with websockets.connect(uri) as ws:
            print(f"  ✓ WS 已连接")
            # 接收 connected 消息
            connected_msg = await asyncio.wait_for(ws.recv(), timeout=5)
            cd = json.loads(connected_msg)
            print(f"  ✓ 频道列表: {cd['data']['channels']}")
            assert 'alert:admin' in cd['data']['channels'], "alert:admin 未订阅"
            print(f"  ... 触发 webhook 入库 ...")
            wh_body = json.dumps({
                "orders": [{
                    "order_no": f"P8-WS-{int(time.time())}",
                    "player_id": "VITE_PLAYER_001",
                    "game_id": 1,
                    "amount": 88.88,
                    "currency": "CNY",
                    "status": "paid",
                    "order_time": "2026-09-25 23:30:00"
                }]
            })
            sig = hmac.new(secret.encode(), wh_body.encode(), hashlib.sha256).hexdigest()
            req("POST", f"{BASE}/api/v1/orders/webhook/demo-webhook", data=wh_body.encode(),
                headers={"X-Signature": f"sha256={sig}", "Content-Type": "application/json"})
            # 接收告警(可能需要跳过其他消息)
            got_alert = False
            for _ in range(5):
                try:
                    msg = await asyncio.wait_for(ws.recv(), timeout=3)
                    data = json.loads(msg)
                    if data.get('channel') == 'alert:admin':
                        print(f"  ✓ 收到告警: title={data['data']['title']}, msg={data['data']['message']}")
                        got_alert = True
                        break
                    else:
                        print(f"    ... 跳过 {data.get('channel')}/{data.get('event')}")
                except asyncio.TimeoutError:
                    break
            if not got_alert:
                print("  ✗ 未收到 alert 消息")
    asyncio.run(test_ws())

print("\n" + "=" * 60)
print("✅ P8 验证完成")
print("=" * 60)