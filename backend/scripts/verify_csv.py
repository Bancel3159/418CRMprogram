#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""P7 CSV upload end-to-end verification."""
import json
import sys
import urllib.request
import urllib.parse
import urllib.error
import os

BASE = "http://localhost:8080"
CSV_PATH = r"E:\MyProject\418CRMprogram\backend\scripts\test_orders.csv"

def post(url, data=None, headers=None):
    if data is None:
        data = b""
    elif isinstance(data, dict):
        data = json.dumps(data).encode()
    req = urllib.request.Request(url, data=data, method="POST", headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        return {"_error": e.code, "body": body}

def get(url, headers=None):
    req = urllib.request.Request(url, headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return {"_error": e.code, "body": e.read().decode()}

def post_multipart(url, fields, file_field, file_path, headers=None):
    boundary = "----FormBoundary123456"
    body = b""
    for k, v in fields.items():
        body += f"--{boundary}\r\n".encode()
        body += f'Content-Disposition: form-data; name="{k}"\r\n\r\n'.encode()
        body += f"{v}\r\n".encode()
    # file
    body += f"--{boundary}\r\n".encode()
    fname = os.path.basename(file_path)
    body += f'Content-Disposition: form-data; name="{file_field}"; filename="{fname}"\r\n'.encode()
    body += b"Content-Type: text/csv\r\n\r\n"
    with open(file_path, "rb") as f:
        body += f.read()
    body += f"\r\n--{boundary}--\r\n".encode()

    h = headers.copy() if headers else {}
    h["Content-Type"] = f"multipart/form-data; boundary={boundary}"
    req = urllib.request.Request(url, data=body, method="POST", headers=h)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        return {"_error": e.code, "body": e.read().decode()}

def main():
    print("=== 1. 登录 ===")
    r = post(f"{BASE}/api/v1/auth/login",
             {"username": "admin", "password": "admin123", "turnstile_token": "dev-bypass"})
    if r.get("_error"):
        print(f"登录失败: {r}"); return
    token = r["data"]["access_token"]
    h = {"Authorization": f"Bearer {token}"}
    print(f"Token OK ({len(token)} chars)")

    print("\n=== 2. 查找模板 ===")
    r = get(f"{BASE}/api/v1/upload-templates?game_id=1", h)
    tpl_id = r["data"][0]["id"]
    print(f"TPL_ID={tpl_id}")

    print("\n=== 3. CSV 上传 ===")
    r = post_multipart(
        f"{BASE}/api/v1/orders/import",
        {"game_id": "1", "template_id": str(tpl_id)},
        "file",
        CSV_PATH,
        h
    )
    if r.get("_error"):
        print(f"上传失败: {r}")
    else:
        s = r["data"]["summary"]
        print(f"批次号: {r['data']['batch']}")
        print(f"总行数: {s['total']}")
        print(f"新增: {s['inserted']}, 跳过: {s['skipped']}, 失败: {s['failed']}")
        print(f"自动归属: {s['auto_attributed']}, 待人工: {s['pending_attribution']}")
        if r['data'].get('row_errors'):
            print("行错误:")
            for err in r['data']['row_errors']:
                print(f"  第{err['row_number']}行 [{err['field']}] {err['message']}")

    print("\n=== 4. 上传后的统计 ===")
    r = get(f"{BASE}/api/v1/orders/stats", h)
    o = r["data"]["overview"]
    print(f"总营收: ¥{o['total_revenue']}")
    print(f"订单数: {o['total_orders']}")
    print(f"付费玩家: {o['paid_players']}")
    print(f"ARPU: ¥{o['arpu']}")
    print(f"A 队业绩: ¥{o['a_team_revenue']} ({o['a_team_orders']} 单)")
    print(f"B 队业绩: ¥{o['b_team_revenue']} ({o['b_team_orders']} 单)")
    print(f"待人工绑定: {o['pending_orders']}")

    print("\n=== 5. 查看最新订单 ===")
    r = get(f"{BASE}/api/v1/orders?page=1&page_size=5", h)
    for o in r["data"]["list"]:
        print(f"  [{o['id']}] {o['order_no']} ¥{o['amount']} src={o['source']} attr={o['attribution_status']} lead={o['lead_id']}")

    print("\n✅ CSV 上传验证完成")

if __name__ == "__main__":
    main()