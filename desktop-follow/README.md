# 桌面端开发说明

## desktop-lead（拉新端）

员工用于提交新线索。

### 启动方式（开发模式，纯 Vite）

```bash
cd desktop-lead
pnpm install
pnpm dev  # http://localhost:5174
```

### Tauri 打包（生产模式）

需要先安装 Rust + Tauri CLI：

```bash
# 安装 Rust（已有）
# 安装 Tauri CLI
cargo install tauri-cli --version "^2.0"

# 启动 Tauri 桌面窗口（同时启 Vite）
cd desktop-lead
cargo tauri dev

# 打包
cargo tauri build
```

### 默认登录

- `a_staff_02 / pass123`（A 部门员工）

---

## desktop-follow（维护端）

员工从公开池抢单，跟进玩家。

### 启动方式（开发模式）

```bash
cd desktop-follow
pnpm install
pnpm dev  # http://localhost:5175
```

### Tauri 打包

```bash
cd desktop-follow
cargo tauri dev
cargo tauri build
```

### 默认登录

- `b_staff_01 / pass123`（B 部门员工）
- `b_manager_01 / pass123`（B 部门主管）

### 核心功能

1. **公开池实时推送**：WebSocket 监听新线索，进入桌面端弹窗提醒
2. **抢单 + 自动复制 token**：点击"抢单"瞬间，token 自动写入剪贴板
3. **玩家详情**：含 13 字段、token 一键复制、流转时间线

---

## 后端依赖

两个桌面端都依赖 Go 后端在 `localhost:8080` 运行。
Vite dev server 已配置 `/api` 代理到后端，`/ws` 代理 WebSocket。

启动后端：

```bash
cd backend
.\bin\server.exe
```