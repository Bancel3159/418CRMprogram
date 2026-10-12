#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import sys
import json
import os
sys.stdout = open(sys.stdout.fileno(), mode='w', encoding='utf-8', buffering=1)
import urllib.request
import urllib.error

BASE = "http://localhost:8080"

def post(url, data, headers=None):
    if isinstance(data, dict):
        data = json.dumps(data).encode()
    req = urllib.request.Request(url, data=data, method="POST", headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        return {"_error": e.code, "body": e.read().decode()}

def get(url, headers=None):
    req = urllib.request.Request(url, headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        return {"_error": e.code, "body": e.read().decode()}

# Login
r = post(f"{BASE}/api/v1/auth/login",
         {"username": "admin", "password": "admin123", "turnstile_token": "dev-bypass"})
token = r["data"]["access_token"]
h = {"Authorization": f"Bearer {token}"}

# Read CSV
csv_path = r"E:\MyProject\418CRMprogram\backend\scripts\test_orders.csv"
with open(csv_path, "rb") as f:
    csv_bytes = f.read()
print(f"CSV bytes: {len(csv_bytes)}")
print(f"First 3 bytes (hex): {csv_bytes[:3].hex()}")
# Strip BOM if present
if csv_bytes[:3] == b'\xef\xbb\xbf':
    csv_bytes = csv_bytes[3:]
    print("BOM detected, stripped")

# Multipart upload with debugging
boundary = "----TestBoundary"
body = b""
for k, v in [("game_id", "1"), ("template_id", "1")]:
    body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"{k}\"\r\n\r\n{v}\r\n".encode()
body += f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"test.csv\"\r\nContent-Type: text/csv\r\n\r\n".encode()
body += csv_bytes
body += f"\r\n--{boundary}--\r\n".encode()

req = urllib.request.Request(
    f"{BASE}/api/v1/orders/import",
    data=body,
    method="POST",
    headers={**h, "Content-Type": f"multipart/form-data; boundary={boundary}"}
)
try:
    with urllib.request.urlopen(req, timeout=30) as r:
        result = json.loads(r.read().decode())
        print("\n=== 上传结果 ===")
        print(json.dumps(result, ensure_ascii=False, indent=2))
except urllib.error.HTTPError as e:
    print(f"HTTP {e.code}: {e.read().decode()}")