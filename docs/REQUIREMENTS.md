# 418 CRM 数据中台 — 需求规格说明书

> **状态**：v1.0 待用户审阅
> **最后更新**：2026-09-25

## 1. 项目背景

公司从事**游戏推广业务**：从游戏公司获得推广权，A 部门员工负责拉新（让新玩家进入游戏），B 部门员工负责维护玩家并引导充值（"氪金"）。**员工在游戏里的身份不是客服，而是"第三方玩家"**，这是本系统的核心业务特征。

## 2. 技术栈

| 层 | 选型 |
|----|------|
| Web 前端 | TypeScript + Vue 3 + Vite + Naive UI + Pinia |
| 桌面端 A/B | Tauri (Rust) + Vue 3 + Vite + Naive UI |
| 后端 | Go 1.22+ + Gin + GORM + PostgreSQL + Redis |
| 实时通信 | WebSocket（gorilla/websocket） |
| 对象存储 | MinIO / S3 兼容存储 |
| 鉴权 | 账号密码 + JWT + Cloudflare Turnstile |
| 部署 | Docker / Docker Compose |

## 3. 角色与权限（RBAC）

| 角色 | 描述 | Web 端访问 | 桌面端 A | 桌面端 B |
|------|------|-----------|----------|----------|
| **admin** | 超级管理员，所有权限 | ✅ 全公司 | ❌ | ❌ |
| **A 部门主管** | A 部门全量数据 + 看板 | ✅ A 部门 | ❌ | ❌ |
| **A 部门组长** | 本组数据 + 看板 | ✅ A 部门（限定本组） | ❌ | ❌ |
| **A 部门员工** | 仅本人提交的线索 | ❌ | ✅ | ❌ |
| **B 部门主管** | B 部门全量数据 + 看板（你） | ✅ B 部门 | ❌ | ❌ |
| **B 部门组长** | 本组数据 + 看板 | ✅ B 部门（限定本组） | ❌ | ✅ |
| **B 部门员工** | 仅本人维护的玩家 | ❌ | ❌ | ✅ |

**核心规则**：
- **Web 端**：仅 admin / 主管 / 组长可登录。员工无 Web 端权限。
- **桌面端 A**：仅 A 部员工（含主管/组长，但通常用 Web）+ admin 调试用
- **桌面端 B**：仅 B 部员工 + 组长（管理本组）
- 主管可调整自己部门的数据（释放/转移），不可跨部门
- admin 可跨部门任意调整

## 4. 组织架构

```
公司
├── A 部门（拉新部）
│   ├── A-组 1
│   │   ├── A 员工 1
│   │   └── A 员工 2
│   └── A-组 2
│       └── ...
└── B 部门（维护部）
    ├── B-组 1
    │   ├── B 员工 1
    │   └── B 员工 2
    └── B-组 2
        └── ...
```

## 5. 核心实体

### 5.1 `user`（用户/员工）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| username | string unique | 登录账号 |
| password_hash | string | bcrypt |
| real_name | string | 真实姓名 |
| role | enum | admin / a_manager / a_lead / a_staff / b_manager / b_lead / b_staff |
| dept_id | FK → department | 所属部门 |
| group_id | FK → department | 所属组（可选） |
| status | enum | enabled / disabled |
| created_at | datetime | |

### 5.2 `department`（部门/组）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| name | string | 部门名 |
| parent_id | FK → department | 上级部门（null = 顶级） |
| type | enum | company / dept_a / dept_b / group |
| created_at | datetime | |

### 5.3 `game`（游戏）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| code | string unique | 缩写（如 TWD） |
| name | string | 全称（如 the walking dead survivors） |
| status | enum | enabled / disabled |
| remark | text | 备注 |
| created_at | datetime | |

### 5.4 `lead`（线索 / 拉新记录，13 字段）

| # | 字段 | 类型 | 说明 |
|---|------|------|------|
| 1 | game_id | FK → game | 游戏（下拉选） |
| 2 | creator_id | FK → user | 拉新员工（**自动取登录者**） |
| 3 | channel | string | 来源渠道（字典） |
| 4 | player_username | string | 玩家用户名（WhatsApp / Discord / Instagram 等渠道账号） |
| 5 | player_game_name | string | 玩家游戏名 |
| 6 | player_id | string | 玩家 ID（**订单匹配主键**） |
| 7 | region | string | 区服 |
| 8 | player_country | string | 玩家国家（字典） |
| 9 | player_age | int | 玩家年龄 |
| 10 | is_target | enum | 是否目标：target / not_target |
| 11 | community | string | 社群 |
| 12 | token_code | string | Discord auth token（A 部员工使用的公司自有账号） |
| 13 | screenshots | string[] | 上传截图（文件路径数组） |

**辅助字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| status | enum | **pending（公开池）/ claimed（已认领）/ archived（已流失）** |
| claimed_by | FK → user | 认领人（B 部员工） |
| claimed_at | datetime | 认领时间 |
| released_by | FK → user | 释放人（主管/admin），archived 时记录 |
| released_at | datetime | 释放时间 |
| extension | JSONB | 扩展字段（后续可加） |
| remark | text | 备注 |
| created_at | datetime | 拉新时间 |

**状态流转**：

```
pending ──(B 员工认领)──► claimed ──(主管释放)──► pending
   │                         │
   │                         └──(流失/无效)──► archived
   │
   └──(A 主管/admin 直接归档)──► archived
```

### 5.6 `payment_source`（充值数据源配置）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| game_id | FK → game | 关联游戏 |
| type | enum | api / webhook / manual（手动上传） |
| config | JSONB | 按 type 不同：api 存 URL+key；webhook 存 secret；manual 存表格模板字段映射 |
| sync_cron | string | 定时任务 cron 表达式 |
| status | enum | enabled / disabled |
| created_at | datetime | |

### 5.7 `order`（充值订单表）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| game_id | FK → game | 来源游戏 |
| **player_id** | string | **玩家 ID（关联 lead.player_id）** |
| region | string | 区服 |
| amount | decimal | 充值金额 |
| recharged_at | datetime | 充值时间 |
| source | enum | api / webhook / manual |
| source_ref | string | 原始订单号（用于幂等去重） |
| raw_data | JSONB | 原始数据（不同游戏字段不同，存原样便于追溯） |
| created_at | datetime | 录入时间 |

**重要**：订单表 **不切分业绩归属**。订单金额完整保留，通过 player_id → lead 关联，分别从 creator_id / claimed_by 两个维度聚合。

### 5.8 字典表（`dict`）

统一管理渠道、国家、社群、区服、跟进方式等枚举值。结构：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| category | string | 字典类型（channel / country / community / region / follow_type） |
| code | string | 代码 |
| label | string | 显示名 |
| sort | int | 排序 |
| status | enum | enabled / disabled |

### 5.9 指标系统（设计要点）

**业绩公式由 admin 配置**，不硬编码。系统提供原子指标 + 公式引擎。

#### 原子指标（系统预定义）

| 指标 | 计算方式 | 适用部门 |
|------|---------|---------|
| 拉新数（lead_count） | lead 表 count（按维度） | A 部门 |
| 付费率（pay_rate） | 已充值 lead 数 / 总 lead 数 | A/B 都看 |
| 充值总额（recharge_amount） | sum(order.amount) | A/B 都看 |
| 充值笔数（recharge_count） | order 表 count | A/B 都看 |
| ARPU（arpu） | 充值总额 / 充值 lead 数 | A/B 都看 |
| 跟进数（follow_count） | follow 表 count（**留白**，P7+） | B 部门 |
| 留存数（retention_count） | X 天后还在的 lead 数（**留白**） | B 部门 |

#### 维度（聚合维度）

- 时间（日 / 周 / 月 / 自定义区间）
- 员工（user）
- 部门（department，含组）
- 游戏（game）

#### 业绩公式（v2 设计，先占位）

`metric_formula` 表（**P8 阶段实现**）：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| target_role | enum | a_staff / b_staff / a_manager / b_manager |
| game_id | FK → game | 适用游戏（null = 全部） |
| formula | text | 公式表达式，如 `lead_count * 5 + recharge_amount * 0.001` |
| enabled | bool | |

admin 在 Web 端配置公式，系统按公式实时计算每个员工的业绩。

## 6. 核心业务流程

### 6.1 拉新 → 公开池 → 认领

```
A 部员工（desktop-lead）登录
   │
   ▼ 填写 13 字段线索表单（含上传截图、token 码）
[Submit Lead]
   │
   ├──► 写入 lead 表（status = pending）
   ├──► 随机抽取一条广播模板 → 渲染（含 token 码占位符）
   ├──► WebSocket 广播 → B 部全员桌面端 B
   │
公开池（status = pending 的全部 lead）
   │
   ▼ B 部员工在桌面端 B 看到新线索提醒（含 token 码）
   │
   ▼ 抢单（POST /api/v1/leads/{id}/claim）
   │
   ├──► 后端：
   │    ├── status: pending → claimed
   │    ├── claimed_by: 当前 B 员工
   │    ├── claimed_at: 当前时间
   │    ├── WebSocket 广播 → 公开池所有其他订阅者移除该线索
   │    └── 响应体返回完整 token_code 字段
   │
   ▼ 桌面端 B 前端处理响应：
   ├── 自动调用 navigator.clipboard.writeText(token_code) → 剪贴板立刻可用
   └── 跳转到玩家详情页 → token 码字段完整显示（带"复制"按钮）
   ▼
玩家绑定到 B 员工，进入维护阶段（B 员工可立即登录 Discord 账号维护新玩家）
```

**公开池规则**：
- 新线索提交 → 立刻广播（含 token 码渲染）
- 认领后从公开池消失
- 不认领则一直在
- 没有过期时间

**token 码安全策略**：
- **全员可见**（不区分角色）：A 部、B 部、admin 都能看到完整 token
- 认领接口响应中 token_code **完整明文返回**（业务需要）
- 数据库明文存储（公司自有 Discord 账号，内部业务凭证，非外部隐私数据）

### 6.2 订单归属与业绩计算（不切分）

```
游戏 API / Webhook / 手动表格上传
   │
   ▼ 订单进入 order 表（完整保留）
   │
   ▼ 通过 order.player_id 匹配 lead.player_id
   │
   ▼ 找到 lead 后：
   ├── lead.creator_id (A 员工) ──► A 部门看板「拉新维度」聚合
   └── lead.claimed_by (B 员工) ──► B 部门看板「维护维度」聚合
   │
   ▼
各看板按维度独立计算：
   ├── A 部门：拉新数 / 玩家充值总额（参考）/ ARPU
   └── B 部门：充值总额（核心业绩）/ 跟进数 / 留存数
   │
   ▼
业绩公式（admin 配置，v2 实现）
```

**核心原则**：订单不切分，每笔金额完整保留；AB 两部门从不同维度看，**数据独立、分开计算**。每个游戏的订单字段不同，通过 `payment_source.config` JSONB 做字段映射。

## 7. 鉴权流程

```
桌面端/Web 端登录页
   │
   ▼ 输入账号 + 密码 + Cloudflare Turnstile token
[POST /api/v1/auth/login]
   │
   ├──► 后端调 Cloudflare `/siteverify` 校验 Turnstile
   ├──► 校验通过 → 查 user 表 → bcrypt 校验密码
   ├──► 签发 JWT（access_token + refresh_token）
   │
   ▼ 后续请求 Header 带 `Authorization: Bearer <token>`
```

## 8. WebSocket 设计

### 8.1 频道

| 频道 | 订阅者 | 事件 |
|------|--------|------|
| `pool:broadcast` | B 部全员 | 新线索进入公开池 |
| `pool:remove` | B 部全员 | 线索被认领，需从本地列表移除 |
| `lead:update:{lead_id}` | 线索相关人 | 状态更新 |
| `user:{user_id}` | 某用户 | 个人通知 |

### 8.2 协议

```json
// 服务端 → 客户端
{
  "channel": "pool:broadcast",
  "event": "lead.created",
  "data": {
    "id": 123,
    "game_code": "TWD",
    "rendered_message": "TWD 新注册！一个美国 25 岁的玩家，刚刚加入了 S1 区，潜力客户，请速速对接！！"
  }
}
```

### 8.3 广播模板（`broadcast_template` 表）

admin 可配置**多条模板**，新线索提交时**随机抽取一条**渲染后推送（避免重复感）。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint PK | |
| name | string | 模板名（管理用） |
| template | text | 模板内容，用 `{字段}` 占位 |
| enabled | bool | 是否启用 |
| weight | int | 抽取权重（默认 1） |
| created_at | datetime | |

**支持的占位符**（来自 Lead 表字段）：
- `{游戏}` `{游戏代码}` `{国家}` `{年龄}` `{区服}` `{渠道}` `{社群}` `{是否目标}` `{玩家游戏名}` `{玩家用户名}` `{token码}` `{token_code}`

> **token 码参与模板渲染**：是公司自有 Discord 账号的 auth token，是 B 部门员工接手工作的必要凭证，必须在广播里就能看到。

**示例模板**：
```
{TWD} 新注册！一个{美国}{25}岁的玩家，刚刚加入了{S1}区，{备注}，请速速对接！！
{游戏} 来活儿啦！{国家}{年龄}岁的{是否目标}玩家，入驻{区服}，{渠道}渠道来的~
...
```

### 8.4 连接方式

- 桌面端登录后建立 WebSocket 连接
- URL：`/ws?token=<jwt>`
- 后端校验 token → 解析 user → 按角色订阅相应频道
  - admin / 主管 / 组长：订阅 `pool:broadcast` / `pool:remove`（只读）
  - B 部员工：订阅 `pool:broadcast` / `pool:remove`（可操作）

## 9. 接口分层

```
/api/v1/
├── auth/                  登录、刷新、登出
├── users/                 用户管理（admin）
├── departments/           部门管理（admin）
├── games/                 游戏管理
├── dicts/                 字典
├── leads/                 线索 CRUD、公开池、认领、释放
├── payment-sources/       充值数据源配置
├── orders/                充值订单（P7 实施）
├── stats/                 看板指标（拉新数、付费率、充值、ARPU 等）
├── broadcast-templates/   广播模板管理（admin）
├── ws/                    WebSocket 连接
└── upload/                文件上传
```

## 10. 看板指标

### B 部门主管（核心用户）

| 指标 | 说明 |
|------|------|
| 今日/本周/本月充值总金额 | 部门业绩核心 |
| B 员工业绩排行 | 我的下属谁做得好 |
| 每个员工的维护玩家数 + 充值金额 | 一对一透明 |
| 各游戏充值贡献 | TWD 占多少、其他游戏占多少 |
| ARPU（人均充值） | 维护质量 |
| 公开池待认领数 | B 部活跃度 |
| A → B 转化漏斗 | 拉新 → 充值 |
| 跟进次数 / 玩家活跃度 | 长期维护 |

### A 部门主管

| 指标 | 说明 |
|------|------|
| 今日/本周/本月拉新数 | 拉新业绩 |
| 各渠道拉新分布 | 渠道质量 |
| 各员工业绩排行 | |
| 拉新 → 充值转化漏斗 | 拉到的人最终充了多少 |

### 管理员 (admin)

- 全公司总览
- 各部门对比
- 系统健康度
- 可切换视角看任意角色

## 11. Admin 后台模块

| 模块 | 说明 |
|------|------|
| 用户管理 | 增删改查、批量导入、重置密码 |
| 部门管理 | A/B 部门 + 子部门树 |
| 角色权限 | 7 个固定角色的权限点配置 |
| 游戏管理 | game 表 CRUD |
| 充值数据源管理 | payment_source CRUD + 测试连接 |
| 字典管理 | 渠道、国家、社群、区服等 |
| 广播模板管理 | broadcast_template 增删改、占位符说明 |
| **业绩公式配置**（P8） | admin 配置各角色业绩公式 |
| 操作日志 | 谁在什么时间做了什么 |

## 12. 留白 / 待定

| 项 | 状态 |
|----|------|
| 跟进记录字段（follow） | 暂不实现，P7+ 后续确定 |
| 业绩公式（admin 配置） | P8 阶段实现，公式系统预留 |
| Discord token 安全 | 明文存储，**全员可见完整值**（公司内部业务凭证，非外部敏感数据） |
| token 码复制机制 | **认领时自动复制到桌面端 B 剪贴板** + 玩家详情页显示完整值（带复制按钮） |
| 跟进提醒规则 | 暂不实现 |
| 多语言 | 暂只支持中文 |
| Web 端「模拟视角」 | admin 是否能模拟任意角色视角看数据，待定 |
| 留存数指标 | 需要时间窗口配置，留白 |

## 13. 实施阶段

| 阶段 | 内容 |
|------|------|
| **P0 基础** | Go 后端骨架 + Docker Compose（PG/Redis/MinIO）+ 配置 + 迁移工具 |
| **P1 鉴权** | User/Department 表 + JWT + Turnstile + RBAC 中间件 |
| **P2 基础数据** | Game / Dict / BroadcastTemplate CRUD |
| **P3 线索核心** | Lead 表 + 13 字段 + 公开池 + 认领 + WebSocket + 广播模板 |
| **P4 Web 端** | 登录 + 主布局 + Lead 管理 + 看板 + 系统管理 + 广播模板配置 |
| **P5 Tauri 桌面端** | desktop-lead（录入线索）+ desktop-follow（抢单 + 维护） |
| **P6 报表** | 看板数据接口（原子指标）+ 各部门看板 |
| **P7 充值** | Order 表 + 数据源接入 + 玩家匹配 + 指标聚合 |
| **P8 业绩公式**（v2） | admin 配置业绩公式 + 实时计算 |

---

**审阅要点**：

1. 实体/字段是否符合业务预期
2. 角色权限是否合理
3. 业务流程是否符合实际工作方式
4. 看板指标是否覆盖你日常关心
5. 实施阶段顺序是否合理

> **请审阅后告诉我**：✅ 通过 / ❌ 哪里要改 / ➕ 哪里要加。通过后我就开始 P0。