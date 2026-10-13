# AGENTS.md — 418 CRM 数据中台 · AI 协作约定

> 给后续接手本项目的 AI 助手(也包括新对话里的自己)准备的导航文件。
> 读完这份文件,你应该能立刻知道:栈是什么、目录怎么分、怎么起服务、有哪些坑别踩、关键架构决策是什么。

---

## 0. 项目一句话

企业 CRM / 数据中台。**三端协同**:web 后台(主管/组长/管理员)+ desktop-lead(拉新员录入线索)+ desktop-follow(维护员跟进)。Go + GORM + PostgreSQL 后端,Tauri 2.x + Vue 3 + Naive UI 前端。

---

## 1. 技术栈(固定,不要轻易换)

| 层 | 技术 |
|----|------|
| 后端 | Go 1.22+ / Gin / GORM / PostgreSQL 16 / Redis 7 |
| 后端鉴权 | JWT(access + refresh)+ Cloudflare Turnstile + RBAC 中间件 |
| 后端日志 | log + 自封装 `pkg/logger` |
| Web 前端 | TypeScript + Vue 3 + Vite + Naive UI + Pinia + Vue Router |
| 桌面端 | Tauri **2.x** + Vue 3 + Naive UI(与 web 共享 design token) |
| 桌面端额外 | `tauri-plugin-global-shortcut`、`tauri-plugin-clipboard-manager`、`tauri-plugin-dialog`、`tauri-plugin-fs`、`global-hotkey = "0.8"` |
| 桌面端系统集成 | `windows-sys = "0.59"`(Win32 DWM API 用于窗口圆角) |
| 部署 | Docker Compose(只跑 PG + Redis,**Go 服务没容器化**) |
| 包管理 | pnpm(web + 两个 desktop) |

> 不要建议换栈。用户多次明确「栈固定」。

---

## 2. 目录结构

```
E:\MyProject\418CRMprogram\
├── backend/                   # Go 后端
│   ├── cmd/
│   │   ├── server/            # 主服务入口
│   │   ├── seed/              # 全量种子
│   │   ├── seed_p7/           # P7 阶段种子
│   │   └── migrate/           # 一次性迁移
│   ├── internal/
│   │   ├── api/               # HTTP handler + 路由
│   │   ├── model/             # GORM 模型 + 枚举
│   │   ├── service/           # 业务逻辑 + cron runner
│   │   ├── config/            # viper + godotenv
│   │   ├── middleware/        # JWT / RBAC / CORS / 日志
│   │   ├── storage/           # 文件上传
│   │   └── ws/                # WebSocket Hub
│   ├── pkg/
│   │   ├── database/          # PG / Redis 连接
│   │   ├── logger/
│   │   └── response/          # 统一响应 {code, message, data}
│   ├── migrations/init.sql    # 一次性建表 + 初始数据
│   ├── scripts/               # dev-start.ps1 / dev-stop.ps1 / verify_*.py
│   ├── bin/server.exe         # cargo build 产物
│   └── .env                   # 后端配置(已纳入项目,不改)
├── web/                       # Web 端(主管后台)
│   ├── src/
│   │   ├── api/modules.ts     # 所有后端 API 类型 + 调用封装
│   │   ├── router/index.ts    # 路由表 + 旧路径重定向
│   │   ├── layouts/MainLayout.vue
│   │   ├── views/system/      # 系统管理页面
│   │   ├── views/leads/       # 线索管理
│   │   └── styles/            # design tokens
│   ├── scripts/dev-start.ps1  # hidden 窗口启 Vite :5273
│   └── scripts/dev-stop.ps1
├── desktop-lead/              # 拉新端
│   ├── src/                   # Vue 前端
│   ├── src-tauri/
│   │   ├── src/lib.rs         # Rust 入口 + 全局快捷键 + DWM 圆角
│   │   ├── tauri.conf.json    # window / bundle / plugin 配置
│   │   └── target/debug/crm-desktop-lead.exe
│   └── scripts/dev-start.ps1 / dev-stop.ps1
├── desktop-follow/            # 维护端(同 desktop-lead 结构)
├── shared/                    # 三端共享的类型(目前主要是文档/规范,代码复用不多)
├── docs/
│   ├── REQUIREMENTS.md        # 业务需求 + ER 表设计
│   ├── ARCHITECTURE.md        # 架构图
│   ├── API.md
│   ├── DATABASE.md
│   └── DEPLOY.md
├── docker-compose.yml         # PG + Redis(Go 服务不在这里)
└── .env                       # docker-compose 用的环境变量(端口/密码)
```

---

## 3. 沟通与代码风格

### 用户偏好
- **语言**:界面/对话中文为主,代码注释英文 OK,变量名英文
- **文件位置**:源码放 workspace 目录,**不要**放到 Desktop/Downloads/home
- **错误反馈**:用户**拒绝**只编译过、要求端到端验证(改了后端一定要打接口 + 看 DB 行;改了前端要看实际渲染)

### 命名/格式
- 后端 Go:用 `gorm` tag + json tag,字段命名 snake_case
- 前端 TS:接口用大驼峰 `interface` 导出在 `web/src/api/modules.ts`,路由用 kebab-case
- 桌面端 Rust:`apply_windows_rounded_corners` 这种 `verb_noun` 风格
- 中文文案:`基础信息 / 是否目标 / 提交注册` 这种短词,不写长句

### PowerShell 特殊事项
- PowerShell 默认 GBK,涉及 `.ts` / `.vue` 写中文 → 文件开头加 **UTF-8 BOM**(`﻿`)
- 不要用 `&&`、Bash 风格 heredoc、`sed -i`、用 `&` 拼接命令 → 全用 PowerShell 原生
- pnpm 走 `cmd /c pnpm` 包装(直接 pnpm 在某些脚本环境会偶发挂)

---

## 5. 服务启动(端口 + 命令)

| 服务 | 端口 | 启动方式 |
|------|------|---------|
| PostgreSQL | 5432 | Docker `crm-postgres`(由 docker-compose) |
| Redis | 6379 | Docker `crm-redis` |
| Go 后端 | **8080** | `backend/scripts/dev-start.ps1`(`bin/server.exe`,改 Go 源码需先 `cargo build` … 哦不,Go 直接 `go build -o bin/server.exe ./cmd/server` 即可) |
| Web 端(Vite) | **5273** | `web/scripts/dev-start.ps1`(原 5173,见 7.1.1) |
| desktop-lead(Vite) | **5274** | `desktop-lead/scripts/dev-start.ps1`(原 5174) |
| desktop-lead(Tauri 窗) | — | dev-start 内部启动 `target/debug/crm-desktop-lead.exe` |
| desktop-follow(Vite) | **5275** | `desktop-follow/scripts/dev-start.ps1`(原 5175) |
| desktop-follow(Tauri 窗) | — | 同上 |

**当前 Docker 状态**:PG + Redis 已起(`docker-compose up -d`)。Go 服务**不在 Docker 里**,必须本地启。

**dev-start 脚本风格**(所有端统一):
- `-WindowStyle Hidden` 启子进程(不弹黑框)
- TCP port 探测(`BeginConnect` + `AsyncWaitHandle`),超时退出非零
- 输出日志到 `.dev-logs/{vite,server,tauri}.log` + `.err.log`
- `Start-Process` 起的子进程脱离控制台组,父脚本退出/被杀不影响
- 注意:**不要**在脚本末尾放 `while ($true) { Start-Sleep 60 }` 这种长驻循环,会被超时机制误杀;脚本验证完成就 `exit 0`
- 杀端口占用用 `Get-NetTCPConnection -LocalPort <port>`,不要用 `$pid` 这个保留变量,改名 `$procId`

### 用户名 / 密码(测试用)

| 账号 | 密码 | 角色 | 层级 |
|------|------|------|------|
| admin | admin123 | 管理员 | 跨全部 |
| beike | test123 | A 部拉新员(a_lead) | 直属下属:晴天/饼干 |
| xingdong | test123 | A 部主管(a_manager) | 兴东 ← 贝壳 |
| qingtian / qingdou | test123 | A 部员工(a_staff) | 兴东 → 晴天 → 贝壳 |
| 等等 | — | — | 详见 DB `users` 表 |

**演示游戏**:`TWD` / `GGW` / `TEST`(TEST 配了 webhook partner,带 HMAC)。

---

## 6. 关键架构决策(踩出来的,别再改回去)

### 6.1 用户管理:上下级 vs 部门

- DB 表 `users` 加了 `superior_id *uint`(自关联,谁是我的直属上级)
- `dept_id` / `group_id` 保留在 DB 但**前端不再展示**,JWT 还在拿
- 权限三层:**admin > 主管(同部门)> 组长(仅直属下属 `superior_id == actorID`)**
- 不允许跨级查看
- `scope=all` 对主管/组长**不开放全部**,只聚合「自己 + 直属下属」;admin 保留全体 A 部
- 文件:`backend/internal/model/user.go`、`backend/internal/api/{user,stats_mine,stats_team}.go`

### 6.2 字典简化

- `dict_category` 砍到只剩 `channel`(用户明确指示)
- 移除:`country` / `community` / `region` / `follow_type`
- 字典分类只用来给 lead 表单的下拉选项供源
- 文件:`backend/internal/model/dict.go`、`backend/internal/api/dict.go`、`web/src/views/system/Dicts.vue`

### 6.3 扩展字段联动

- `lead_extension_fields.type = 'select'` 支持:**自定义 JSON 数组** 或 **引用字典分类(`dict_category`)**
- 引用时后端实时把字典条目塞到响应里的 `options` 字段,前端不用再二次请求
- 引用字典分类只允许 `channel`
- 文件:`backend/internal/model/lead_extension_field.go`、`backend/internal/api/extension_field.go`、`web/src/views/system/ExtensionFields.vue`

### 6.4 菜单重组(单页面 + tabs)

- **配置中心** = 单页 `ConfigCenter.vue`,tabs 嵌入 `Dicts` + `ExtensionFields`
- **游戏管理** = 单页 `GamesCenter.vue`,tabs:游戏列表 + 上传模板(原 `合作方管理` tab 删了)
- **游戏列表** `Games.vue` 内嵌「数据回传」(webhook / API 拉取)分组,sheet 全展开
- 人设库:已删除(2025-10-02)—— DB `personas` 表也 DROP 了
- 合作方管理:完全内联到 Game CRUD,`Partners.vue` + `partnerApi` 删除
- 广播模板:从独立菜单移到 `公开池(Pool.vue)` 的 tabs 里,admin 可见
- 文件:`web/src/router/index.ts`、`web/src/layouts/MainLayout.vue`、`web/src/views/system/*.vue`、`web/src/views/leads/Pool.vue`

### 6.5 合作方内联到 Game

- `games` 表加 `partner_type *string` + `partner_config JSONB` + `partner_enabled bool`(计算字段)
- `partner_type` ∈ `{webhook, api, none}`,webhook 时 `partner_config` 存 `hmac_secret` + `field_mappings`
- webhook 入口 `/api/v1/orders/webhook/:code` 查 `games WHERE code=? AND partner_type='webhook'`,不再查 `partners` 表
- 旧的 `partners` / `partner_type` 表保留但前端不再用
- 文件:`backend/internal/model/game.go`、`backend/internal/api/{game,webhook}.go`

### 6.6 Lead 端快捷键:自由录制

- 前端 `Settings.vue` 用户按键组合 → 字符串(`Ctrl+\`` / `Ctrl+Shift+K` …)
- 存 localStorage `lead_hotkey_combo`,默认值 `Ctrl+\``
- `invoke('set_window_shortcut', { combo })` → Rust `parse_shortcut()` → `global-shortcut` crate
- Rust 强制**至少 1 个修饰键**(防误触)
- 接收 `disabled` / `none` / `""` 表示注销
- 文件:`desktop-lead/src/{App.vue,views/Settings.vue}`、`desktop-lead/src-tauri/src/lib.rs`、`desktop-lead/src-tauri/Cargo.toml`

### 6.6.1 JWT TTL(三端统一)

| 配置 | 含义 | 当前值 |
|------|------|--------|
| `CRM_JWT_ACCESS_TTL` | access token 有效期 | **86400s = 24h**(2026-10-03 改) |
| `CRM_JWT_REFRESH_TTL` | refresh token 有效期 | 604800s = 7d |

- 三端共用同一份 JWT 配置(web / lead / follow),改 `.env` 全端生效
- **三端当前都没有自动 refresh 逻辑** — `refresh_token` 存在 localStorage / store,但前端 401 拦截器直接清掉跳登录页,从来不主动调 `/auth/refresh`
- 因此 access 过期 = 立即被踢;refresh_token 仅在用户主动重新登录时被后端校验
- 如果想要"7 天内永不踢出",需要前端在 401 时主动调 `/auth/refresh` 续 access(暂未做)
- 文件:`backend/.env`、`backend/internal/config/config.go`

### 6.7 Lead 端窗口圆角(DWM API)

- `tauri.conf.json`:`decorations:false + transparent:true + shadow:false`
- OS 窗口**用 DWM API** `DwmSetWindowAttribute(DWMWA_WINDOW_CORNER_PREFERENCE = DWMWCP_ROUND)` 强制圆角
- 前端 `#app { border-radius:14px; overflow:hidden; transform:translateZ(0); }`(transform 让 #app 成为 fixed 子元素的 containing block,这样 overflow 才能裁剪 WindowControls)
- `box-shadow` 用 CSS 自己造视觉深度(替代 OS 阴影)
- `WindowControls.vue` 关闭按钮 `border-radius: 0 14px 0 0` 同步窗角
- `MainLayout.vue` header `padding: 0 90px 0 18px`,用户名 tag 推到控件左侧

### 6.8 删了的东西(回收站可恢复,别再加回来)

- `web/src/views/system/Personas.vue`(人设库)
- `web/src/views/system/Partners.vue`(合作方)
- `web/src/views/leads/List.vue`(线索管理改 LeadsCenter + Pool Panel)
- `web/src/views/leads/Pool.vue`(同上)
- `backend/internal/api/persona.go`
- `backend/internal/model/persona.go`
- DB `personas` 表(DROP TABLE)

### 6.9 线索管理 = 三 tabs 单页

```
/leads                  → LeadsCenter.vue (三 tabs 外框)
├── tab1 线索列表         → ListPanel.vue   (原 List.vue 抽离)
├── tab2 线索池           → PoolPanel.vue   (原 Pool.vue「线索池」tab 抽离)
└── tab3 广播模板         → BroadcastPanel.vue (复用 BroadcastTemplates.vue,admin only)

旧链接 /pool → 重定向 /leads?tab=pool
菜单里「公开池」独立菜单项已去掉
```

### 6.10 列表字段可配置(P10 多维表视图机制)

类比 WPS 多维表:同一份数据源(玩家 / 订单)可配置多个视图,每个视图独立选择/排列字段。后端数据模型支持多视图,**MVP 阶段 UI 只编辑每个 entity 的「默认视图」**(每个 entity 一个 is_system + is_default 的记录)。

- DB:`list_views` 表(`model.GetAllModels()` 自动迁移)
- 数据模型:`id, entity_type('player'|'order'), name, is_default, is_system, fields(JSONB), created_by, created_at, updated_at, deleted_at`
- 字段配置结构:`{code, label, source, width, align, formatter}`
- API:
  - `GET /api/v1/list-views?entity_type=X[&is_default=true]` — 默认视图完整 fields(任何登录用户可读,列表页用)
  - `GET /api/v1/list-views/:id` — 单视图
  - `GET /api/v1/list-views/available-fields?entity_type=X` — 字段池(含 extension / raw_payload 动态字段)
  - `POST/PUT/DELETE /api/v1/list-views/...` — admin CRUD
- UI:
  - 配置中心 4 tabs(字典/扩展字段/**玩家列表字段配置**/**订单列表字段配置**)
  - 「玩家列表字段配置」「订单列表字段配置」共用 `web/src/components/ListViewEditor.vue`
  - 字段池按 source 分组(拉新端核心/聚合/时间/维护端跟进/订单聚合/自定义扩展等)
  - 已配置字段支持 ↑↓ 排序、✎ 编辑(显示名/列宽/对齐/formatter)、✕ 移除
  - 顶部「保存」+「放弃改动」,有 dirty 标记
- 列表页消费:Players.vue / orders/List.vue 启动时拉默认视图,`computed` 动态构建 `columns`,formatter 派发(tag / money / datetime / number / user_list 等)
- 种子:cmd/seed 启动时若发现 player/order 默认视图不存在则创建(11 字段,继承旧的写死列宽)
- 字段池覆盖:
  - player:lead_basic(9) + lead_aggregation(3) + lead_time(2) + follow_record(4) + order_summary(5) + extension.*(动态)
  - order:order_core(11) + order_time(4) + lead_joined(7) + order_meta(4) + raw_payload.*(动态)

### 6.11 follow 端实时通知(P12 订单 / 玩家激活)

需求 3:follow 端特殊实时提醒 —— 付费订单入库即时推 + 充值 ≥ 阈值的高额订单独立 event。

**后端**
- `system_configs` 加 2 个 key(常量在 `model/system_config.go`):
  - `realtime.high_value_amount` 默认 `100`(元,`>=` 视为高额)
  - `realtime.enabled` 默认 `true`(`"false"` 一键停推所有 realtime 推送,异常时紧急开关)
- 启动 `service.EnsureRealtimeDefaults(db)` 写默认值;`service.LoadRealtimeConfig(db)` 每次 webhook 实时读(改完下次推送立即生效,不需要重启进程)
- `webhook.go` 在 status=paid 订单入库(`isNew`)后:
  - 推 `order.paid` 或 `order.high_value`(按阈值二选一)到 `follow:realtime` channel
  - 可选字段 `player_last_active_at`(RFC3339):携带则调 `service.UpdateLeadActiveTime` 更新活跃时间 + 若 is_lost 则自动 mark-active
- `ws.Hub.BroadcastToFollowRoles(channel, msgType, event, data)`:循环调 `BroadcastToRole` 3 次(`b_manager` / `b_lead` / `b_staff`);**admin 不收**(他走 web 端全局订单列表)
- WS 连接时 follow 角色自动订阅 channel `follow:realtime`(在 `api/lead.go` 的 wsConnect 里按 `isFollowRole(claims.Role)` 加)
- `POST /api/v1/players/:player_id/active` 单独端点(任意登录用户可调):
  - Body:`{ game_id 必填, event_time 可选, source 'manual'|'webhook'|'api' }`
  - 行为:`UpdateLeadActiveTime` + 推 `player.react` 事件到 follow 角色(供手动标记 / 合作方推)
- API:`GET|PUT /api/v1/system/configs/realtime`(GET 任意登录用户可读,PUT 仅 admin)
- 文件:`backend/internal/model/system_config.go`、`backend/internal/service/realtime.go`、`backend/internal/ws/hub.go`、`backend/internal/api/{webhook,follow_active,system_config,router}.go`

**前端**
- `stores/ws.ts`:`RealtimeNotification[]` 队列(上限 20)+ `unreadCount` + localStorage 持久化(key `realtime_notifications_v1`)+ `markAllRead` / `clearAll`;`pushRealtimeNotification` 收到 `follow:realtime` 事件时构造通知项 + 触发 `fireTauriSystemNotification`(见 6.12)
- `components/NotificationBell.vue`:n-badge 数字(>0 才显示)+ 下拉面板(标题 / 正文 / 相对时间)+ 高额项左边框(`border-left: 3px solid #f59e0b`);未读项淡蓝底;点击跳对应详情页
- `MainLayout.vue`:`order.high_value` → 8s 黄色 warning toast;`player.react && was_lost` → 6s 绿色 success toast;两个 toast 点击都跳详情
- `LeadDetail.vue`:watch `wsStore.lastMessage`,命中当前 lead 时自动 `loadDetail()`(不重复弹 toast)
- ⚠️ follow 端详情路由是 `/leads/:id`(不是 `/lead/:id`),通知项的 `link` 字段要写对
- 文件:`desktop-follow/src/stores/ws.ts`、`desktop-follow/src/components/NotificationBell.vue`、`desktop-follow/src/layouts/MainLayout.vue`、`desktop-follow/src/views/LeadDetail.vue`

**消息 payload 形态**
```json
{
  "type": "alert", "channel": "follow:realtime", "event": "order.high_value",
  "data": {
    "order_id": 37, "order_no": "WH-RT-005", "amount": 250,
    "player_id": "wh_test_player_b", "game_id": 4, "game_code": "TEST",
    "lead_id": 9, "claimed_by": null, "claimed_by_name": null,
    "high_value": true, "high_value_threshold": 100
  }
}
```

### 6.12 follow 端 Tauri 系统通知(P12.5 任务栏闪 + 系统 toast)

需求:像微信 / QQ 一样 —— 任务栏窗口变黄、图标闪、弹系统 toast。**仅 Tauri 桌面端生效**(crm-desktop-follow.exe);浏览器 / 纯 WebView 走 6.11 的顶部铃铛即可,不申请 Web Notification 授权。

**Tauri Rust**(`desktop-follow/src-tauri/`)
- `Cargo.toml` 加 `tauri-plugin-notification = "2"` + 平台限定 `windows-sys = { version = "0.59", features = ["Win32_UI_WindowsAndMessaging", "Win32_Foundation"] }`
- `run()` 里 `.plugin(tauri_plugin_notification::init())`
- command `flash_window(window: WebviewWindow, on: bool)`:
  - Win32 `FlashWindowEx` + `FLASHW_TRAY | FLASHW_TIMERNOFG`(闪任务栏,且即使窗口已获焦也继续闪)
  - `uCount = 0` → 由系统控制闪烁直到窗口前台化
- command `show_toast(app, title, body)`:`app.notification().builder().title().body().show()`(Win10/11 走 Action Center)
- `capabilities/default.json` 加 `"notification:default"` 权限
- 文件:`Cargo.toml`、`src/lib.rs`、`capabilities/default.json`

**前端(JS)**(在 `stores/ws.ts` 的 `fireTauriSystemNotification`)
- `isTauri()`:`typeof window !== 'undefined' && !!(window as any).__TAURI_INTERNALS__`
- 收到 realtime 事件 → `invoke('flash_window', { on: true })` + `invoke('plugin:notification|notify', { options: { title, body } })`
- 10 秒后自动 `invoke('flash_window', { on: false })` 强制停止(防止窗口离开前台时无限闪)

**触发范围**:`order.paid` / `order.high_value` / `player.react` **全事件都闪 + toast**(用户 2026-10-04 拍板;若后续觉得吵,收敛到仅 `order.high_value` 即可,改 `pushRealtimeNotification` 的调用点)

### 6.13 玩家全字段强制编辑(P13 玩家流转 / 数据调整)

需求:后期要流转玩家或调整数据时,需要能改**所有字段**,包括常规业务流程不允许动的那些(改游戏、改拉新员、改维护员、改状态机、改付费标记、改流失状态)。

**核心决策**
- **入口**:web 端玩家列表新增「操作」列(固定右侧),仅 admin 可见(`userStore.isAdmin`)
- **定位方式**:`player_id` 是唯一主键(同一玩家进不同游戏 ID 也不同),列表接口返回 `latest_lead_id`,前端直接用它打开编辑弹窗
- **权限**:路由级 `middleware.RequireAdmin()`,非 admin 调接口直接 403
- **审计**:每次编辑写 `audit_logs`(action=`update`, target_type=`lead`, extension 里带 `force_edit: true` + `changed_fields` 精确 diff)

**后端**(`backend/internal/api/lead_force_edit.go`)
- `PUT /api/v1/leads/:id/force-edit`
- `ForceEditReq` 所有字段都是 `*T` 指针 —— **nil = 本次不改**,这样能区分「不改」和「改成零值」
- **开放**:game_id / creator_id / claimed_by / released_by / channel / status / player_id / player_username / player_game_name / region / player_country / player_age / is_target / community / token_code / screenshots / remark / extension / paid_mark / is_lost / lost_reason / reactivated_at / last_player_active_at / 各 `_at` 时间字段
- **不开放**:created_at / updated_at / deleted_at / paid_auto / is_paid(系统生成)
- **仍保留的基本校验**(防止脏数据):游戏必须存在、status/is_target 枚举合法、player_age 0-150
- `diffLeadFields(before, after)` 逐字段比对,只把**真正变化的**字段名写进审计
- **派生字段约定**:`lost_at` 和 `paid_marked_at` 由后端自动维护(`is_lost` 变 true 时写 now;`paid_mark` 变更时写 now/置 NULL),**不进 diff 列表**

**前端**
- `web/src/views/players/PlayerEditDrawer.vue`:720px 右侧抽屉,分 4 组(归属与流转 / 核心业务字段 / 付费与流失状态 / 扩展字段 JSON),底部实时显示「本次将修改的字段 (N)」tag,提交前弹二次确认
- `web/src/views/players/Players.vue`:`dynamicColumns` computed 里判断 `userStore.isAdmin` 才 push 操作列
- `web/src/api/modules.ts`:`leadApi.forceEdit(id, data)`
- ⚠️ **前端 diff 必须排除派生字段**:表单里没有 `lost_at` / `paid_marked_at` 的输入控件,如果直接按字段名遍历比较,会因为 `undefined` != 原值而误报成「已修改」,导致按钮显示"保存修改(2)"但实际提交 0 个字段。用 `DERIVED_FIELDS` Set 显式排除

---

## 7. 踩过的坑(下次别再栽)

### 7.1 Rust 改动后必须手动 cargo build
`dev-start.ps1` 启动的是 `pnpm dev` 只起 Vite,**不会**自动 rebuild Rust。Rust 源码改了必须先 `cd src-tauri && cargo build`,然后**重启 Tauri exe**(旧的 exe 锁着,build 会报 `os error 5 拒绝访问`,要先 Stop-Process 再 build)。

**⚠️ 改 `tauri.conf.json` 的 `devUrl` 同样必须重新 `cargo build`** —— `devUrl` 是编译期被 `tauri::generate_context!()` 烧进 exe 的常量,不是运行时读文件。改完配置不编译 → Tauri 窗口去连旧端口 → 显示「localhost 拒绝连接」。
- 对照:`vite.config.ts` 的 `port` 是 Vite **运行时**读,HMR 直接生效,不用编译
- 2026-10-13 踩过:Vite 从 5175 迁到 5275(Windows 排除端口范围 5041-5240 吞了 5173-5175),改了 conf 但没编译,两端窗口全白屏报拒绝连接
- 验证手段:`[IO.File]::ReadAllBytes(exe)` 然后 `Select-String 'localhost:\d{4}'`,能直接看到烧进去的端口

### 7.1.1 当前端口(2026-10-13 起)

| 服务 | 端口 | 说明 |
|------|------|------|
| PostgreSQL | 5432 | Docker crm-postgres |
| Redis | 6379 | Docker crm-redis |
| Go 后端 | **8080** | `backend/scripts/dev-start.ps1` |
| web 主管后台 | **5273** | 原 5173 |
| desktop-lead 拉新端 | **5274** | 原 5174 |
| desktop-follow 维护端 | **5275** | 原 5175 |

**为什么从 517x 迁到 527x**:Windows 排除端口范围(Hyper-V / WSL / Docker Desktop 自动保留)`5041-5240` 把 5173/5174/5175 全吞了,Vite bind `0.0.0.0` 时报 `EACCES: permission denied`。
- 查命令:`netsh interface ipv4 show excludedportrange protocol=tcp`
- 不要试图改这个系统设置(要管理员权限且影响全局),直接换端口段
- 改了端口要同步 4 处:`{web,desktop-lead,desktop-follow}/vite.config.ts` + 三个 `scripts/dev-start.ps1` 的 `$VitePort` + `{desktop-lead,desktop-follow}/src-tauri/tauri.conf.json` 的 `devUrl`(后者还要 cargo build)

### 7.1.2 dev-start.ps1 三个坑(2026-10-13 修)

1. **不要用 `$pid`** —— PowerShell 保留变量(只读)。`Stop-Occupying()` 里 `$pid = $c.OwningProcess` 直接抛 `VariableNotWritable`,脚本在「Cleaning port」这步就崩,端口没清、Vite 也没重启。用 `$procId`。
2. **末尾不要放 `while ($true) { Start-Sleep 60 }`** —— 前台跑终端不返回、没有 `[OK]` 提示,看起来像卡死。子进程本就是 `-WindowStyle Hidden` detached,脚本 `exit 0` 即可。
3. **写 .ps1 里的中文/特殊符号** —— PowerShell 默认 GBK,em dash(`—`)会显示成 `鈥?`。文件头加 UTF-8 BOM 或直接用 ASCII 字符。

### 7.2 PowerShell `$pid` 是保留变量
`$pid` 是当前进程 PID,**不能赋值**。需要拿 owning PID 时用 `$procId` 之类自定义名。

### 7.3 `Start-Process` 子进程不自动脱离
直接 `Start-Process xxx.exe -WindowStyle Hidden` 起的子进程绑在父控制台组里。**父进程被杀时子进程一起死**。
- 想脱离:用 `cmd /c start "" /B <exe>`(完整脱离)
- 或确保父脚本 `exit 0` 立即退出(server.exe 自动成为孤儿)
- 后端 `dev-start.ps1` 不要写 watcher 长驻循环

### 7.4 windows-sys 0.59 常量是 i32
`DwmSetWindowAttribute` 收 `u32`,但 windows-sys 0.59 把 `DWMWA_WINDOW_CORNER_PREFERENCE` / `DWMWCP_ROUND` 声明成 `i32`。
**修法**:手动用裸值 `let attribute: u32 = 33; let preference: u32 = 2;`

### 7.5 Vite HMR 不刷新 Tauri WebView 缓存
Tauri WebView2 偶尔缓存了旧 CSS。**强刷** = Ctrl+R / F5 / 右键 Reload。

### 7.6 backend/.env 密码
DB 默认 `crm_password`(跟 docker-compose 默认对齐)。改 docker-compose 端口要同步改 backend/.env。

### 7.7 菜单权限
`web/src/router/index.ts` 有 `adminOnly` meta 守卫。**新菜单如果只给管理员**,用 child 标签包。

### 7.8 E: 盘 NTFS 单 inode 偶发损坏(os error 433)

症状:`Get-FileHash` / `[IO.File]::ReadAllBytes` / cargo 写 .rlib 报 `A device which does not exist was specified`(os error 433),但同目录其他文件读写完全正常。

- 2026-10-04 / 10-05 在 `desktop-follow/src-tauri/target/debug/.fingerprint/deranged-*` 和 `crm-desktop-follow.exe` 上都遇到
- **重命名修复不了** —— 损坏的是 inode/MFT 记录本身,rename 后 inode 跟着走,换个新名字指向同一条坏记录照样写不进
- 删除重建后写入正常(cargo 会自己重算 fingerprint);但如果是已存在的文件(exe)删了重拷可能仍读到坏记录
- `git init` 提交后这个项目在 GitHub 上有完整副本,极端情况可 `git clone` 恢复
- **应对**:cargo 一直报同一个 crate 的同一个 fingerprint 路径失败 → 别死磕清理,直接把 `src-tauri` 拷到别的盘(如 `D:\crm-follow-build`)`cargo build`,产物 exe 从那边启动
- 这不是代码问题,也不是某个 crate 的问题(desk-lead 的 E: target 也偶发),纯属 E: 盘 NTFS 偶发损坏

### 7.9 git 仓库已初始化(2026-10-13)

- remote:`git@github.com:Bancel3159/418CRMprogram.git`(SSH,认证身份 `Bancel3159`)
- 本仓库 `user.name` / `user.email` 已设成 `Bancel3159` / `315985660@qq.com`(覆盖全局的 `JD Tool Dev`)
- 首推 commit:`74949ca` / 218 files / 60239 行
- `.gitignore` 已挡住:`.env`(根 + backend)、`backend/bin/`、`node_modules/`、`dist/`、`src-tauri/target/`、`*.log`、`.dev-logs/`、`backend/uploads/`、根目录调试残留(`drawer_check.txt` / `players*.json` / `players_check.txt`)
- 入库的 `.env` 只有 `backend/.env.example`(模板,无真实密钥)
- 克隆后要跑起来:`cp backend/.env.example backend/.env` → 改 `CRM_JWT_SECRET` / `CRM_TURNSTILE_SECRET`

---

## 9. 测试数据约定

新建数据:
- 路由变动 → 老 1974 直跳看板(若需要)统一跳看板
- config?tab=扩展
- pool?tab=broadcast
- system/games?tab=upload / ?tab=partners

文件:`web/src/router/index.ts`

---

## 10. 新对话开局模板(给你或下一个 AI 看)

如果你是在新对话接手这个项目,**先读这份 AGENTS.md**,然后按这个顺序探查:

```
1. E:\MyProject\418CRMprogram\AGENTS.md           ← 你现在在读
2. E:\MyProject\418CRMprogram\README.md           ← 顶层概览
3. backend\.env + docker-compose.yml              ← 当前 DB / Redis 端口
4. Get-Process crm-desktop-lead, server, vite 等  ← 当前哪些服务在跑
5. git status / 用户最近文件                       ← 知道最近改了什么
6. 跟用户确认下一个任务                            ← 业务方向
```

不要假设服务都还在跑 —— 必须实测(Get-NetTCPConnection / Get-Process)。

---

## 11. 关键命令速查

```powershell
# 起后端
cd E:\MyProject\418CRMprogram\backend
.\scripts\dev-start.ps1
.\scripts\dev-stop.ps1

# 起 web
cd E:\MyProject\418CRMprogram\web
.\scripts\dev-start.ps1
.\scripts\dev-stop.ps1

# 起 desktop
cd E:\MyProject\418CRMprogram\desktop-lead   # 或 desktop-follow
.\scripts\dev-start.ps1
.\scripts\dev-stop.ps1

# 重新编译 Go 后端
cd E:\MyProject\418CRMprogram\backend
go build -o bin\server.exe .\cmd\server

# 重新编译 Tauri(改了 Rust 之后)
$env:Path = 'C:\Users\bancel\.cargo\bin;' + $env:Path
cd E:\MyProject\418CRMprogram\desktop-lead\src-tauri   # 或 desktop-follow
cargo build
# 然后 Stop-Process + 重启 dev-start.ps1(或直接启 .exe)

# 看 DB 表
docker exec crm-postgres psql -U crm -d crm_db -c "\dt"

# 改 .env 后需要重启后端 / Vite / Tauri
```

---

**最后更新**:2026-10-13 · P13 玩家全字段强制编辑(6.13)+ P12 实时通知(6.11/6.12)+ git 仓库初始化(7.9)+ 端口迁移 517x→527x(7.1.1)。