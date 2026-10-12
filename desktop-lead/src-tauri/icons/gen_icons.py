#!/usr/bin/env python3
# 生成 Tauri 兼容的真正 .ico (PNG-embedded 格式,Vista+ 兼容)
import struct, os, sys, zlib

OUT_DIR = sys.argv[1] if len(sys.argv) > 1 else "."
os.makedirs(OUT_DIR, exist_ok=True)


def make_png(w, h, color=(0x18, 0xa0, 0x58)):
    raw = bytearray()
    for y in range(h):
        raw.append(0)  # filter byte (None)
        for x in range(w):
            r = min(255, color[0] + (x * 256 // w))
            g = min(255, color[1] + (y * 256 // h))
            b = color[2]
            raw.extend([r, g, b, 0xff])

    def chunk(t, d):
        crc = zlib.crc32(t + d)
        return struct.pack(">I", len(d)) + t + d + struct.pack(">I", crc)

    sig = b'\x89PNG\r\n\x1a\n'
    ihdr = struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0)
    idat = zlib.compress(bytes(raw))
    return sig + chunk(b'IHDR', ihdr) + chunk(b'IDAT', idat) + chunk(b'IEND', b'')


def make_png_ico(sizes, color=(0x18, 0xa0, 0x58)):
    """生成 PNG-embedded 格式的 ICO(支持现代 Win 资源编译器)"""
    # 先准备所有 PNG 数据
    pngs = []
    for w, h in sizes:
        pngs.append(make_png(w, h, color))

    # ICO header
    ico_header = struct.pack("<HHH", 0, 1, len(sizes))

    # 目录项
    offset = 6 + 16 * len(sizes)
    entries = bytearray()
    images = bytearray()
    for (w, h), png in zip(sizes, pngs):
        entry = struct.pack(
            "<BBBBHHII",
            w if w < 256 else 0,
            h if h < 256 else 0,
            0, 0, 1, 32,  # reserved, planes, bit count
            len(png),
            offset
        )
        entries.extend(entry)
        offset += len(png)
        images.extend(png)

    return ico_header + bytes(entries) + bytes(images)


# PNG-embedded ICO
sizes = [(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
ico_data = make_png_ico(sizes)
with open(os.path.join(OUT_DIR, "icon.ico"), "wb") as f:
    f.write(ico_data)

# PNG 备用
for name, w, h in [
    ("32x32.png", 32, 32),
    ("128x128.png", 128, 128),
    ("128x128@2x.png", 256, 256),
    ("icon.png", 512, 512),
]:
    with open(os.path.join(OUT_DIR, name), "wb") as f:
        f.write(make_png(w, h))

print(f"Icons generated in {OUT_DIR}")
for f in sorted(os.listdir(OUT_DIR)):
    size = os.path.getsize(os.path.join(OUT_DIR, f))
    print(f"  {f:30} {size} bytes")