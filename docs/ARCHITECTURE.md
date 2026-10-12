# 418 CRM 数据中台 — 整体架构

## 1. 项目定位

面向销售/客户管理场景的数据中台，三端协同：

| 端 | 角色 | 主要职责 |
|----|------|----------|
| **Web 端** | 主管及以上 | 全量数据看板、线索分配、报表分析、权限管理 |
| **桌面端 A（lead）** | 一线员工 | 提交新线索、录入客户信息 |
| **桌面端 B（follow）** | 跟进人员 | 线索跟进、客户维护、签单/回款记录 |

## 2. 技术栈

| 层 | 选型 | 说明 |
|----|------|------|
| Web 前端 | TypeScript + Vue 3 + Vite + Naive UI + Pinia + Vue Router | Composition API + `<script setup>`，Naive UI 做中后台组件 |
| 桌面端 A/B | Tauri (Rust 核心) + Vue 3 + Vite + Naive UI | 复用 web 技术栈，安装包小、启动快 |
| 后端 | Go 1.22+ + Gin / Fiber + GORM / sqlx + JWT | RESTful + 后期可加 WebSocket |
| 数据库 | PostgreSQL（主） + Redis（缓存/会话） | PG 适合复杂查询与统计 |
| 对象存储 | MinIO / 阿里 OSS | 附件、合同图片 |
| 部署 | Docker + Docker Compose / Nginx | 后续上 K8s |

## 3. Monorepo 目录结构

```
418CRMprogram/
├── backend/                # Go 后端 API 服务
│   ├── cmd/server/         # 入口
│   ├── internal/
│   │   ├── api/            # 路由 + Handler
│   │   ├── service/        # 业务逻辑
│   │   ├── repository/     # 数据访问层
│   │   ├── model/          # 领域模型
│   │   ├── middleware/     # JWT、CORS、日志
│   │   └── job/            # 定时任务（统计、提醒）
│   ├── pkg/                # 公共工具（JWT、响应封装、错误码）
│   ├── config/             # 配置加载
│   ├── migrations/         # 数据库迁移（golang-migrate / atlas）
│   ├── docs/               # Swagger 文档
│   ├── go.mod
│   └── Dockerfile
│
├── web/                    # Web 端（主管后台）
│   ├── src/
│   │   ├── api/            # 接口封装
│   │   ├── views/
│   │   │   ├── dashboard/  # 数据看板
│   │   │   ├── leads/      # 线索管理
│   │   │   ├── customers/  # 客户管理
│   │   │   ├── follow/     # 跟进记录
│   │   │   ├── contracts/  # 合同/回款
│   │   │   ├── analytics/  # 报表分析
│   │   │   ├── system/     # 用户/角色/权限
│   │   │   └── login/
│   │   ├── components/     # 业务组件
│   │   ├── layouts/        # 布局
│   │   ├── router/
│   │   ├── stores/         # Pinia
│   │   ├── types/          # 共享 TS 类型（可来自 shared）
│   │   ├── utils/
│   │   └── styles/
│   ├── public/
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── desktop-lead/           # 桌面端 A：提交线索
│   ├── src/                # Vue 前端
│   ├── src-tauri/          # Rust 核心
│   │   ├── src/
│   │   ├── tauri.conf.json
│   │   └── Cargo.toml
│   └── package.json
│
├── desktop-follow/         # 桌面端 B：跟进维护
│   ├── src/
│   ├── src-tauri/
│   └── package.json
│
├── shared/                 # 三端共享
│   └── types/              # TS 类型定义（线索、客户、跟进等）
│
├── docs/
│   ├── ARCHITECTURE.md     # 本文件
│   ├── API.md              # 接口规范
│   ├── DATABASE.md         # 数据库设计
│   └── DEPLOY.md           # 部署文档
│
├── docker-compose.yml      # 一键启动（PG + Redis + MinIO + 后端）
├── .gitignore
└── README.md
```

## 4. 业务流程

```
员工 A (desktop-lead)
   │  提交线索
   ▼
[线索池] ──► 主管 (web) 分配 ──► 员工 B (desktop-follow) 跟进
                                          │
                                          ▼
                                      客户 → 合同 → 回款
                                          │
                                          ▼
                                    主管 (web) 看统计看板
```

## 5. 角色与权限（RBAC）

| 角色 | 桌面端 A | 桌面端 B | Web 端 |
|------|---------|---------|--------|
| 员工（线索录入） | ✅ | ❌ | 仅看自己 |
| 员工（跟进） | ❌ | ✅ | 仅看自己 |
| 主管 | ❌ | ✅ | 全量 |
| 管理员 | ❌ | ❌ | 全量 + 系统设置 |

## 6. 核心实体（领域模型草图）

- **User**：账号、角色、所属部门
- **Lead（线索）**：姓名、电话、来源、状态、归属人
- **Customer（客户）**：从线索转化而来，含详细资料
- **Follow（跟进记录）**：客户 ID、跟进内容、跟进方式、下次提醒
- **Contract（合同）**：客户 ID、金额、产品、生效时间
- **Payment（回款）**：合同 ID、金额、回款日期、责任人
- **Department**：组织架构

## 7. 接口分层约定

```
/api/v1/
├── auth/          # 登录、刷新 token
├── leads/         # 线索 CRUD、分配
├── customers/     # 客户
├── follows/       # 跟进记录
├── contracts/     # 合同
├── payments/      # 回款
├── stats/         # 看板数据
├── users/         # 用户管理
└── system/        # 字典、配置
```

## 8. 下一步建议

1. 确认业务细节（线索来源字典、跟进状态机、回款规则等）
2. 初始化 Go 后端骨架 + 鉴权
3. 初始化 Web 端骨架 + 登录 + 主布局
4. 初始化两个 Tauri 桌面端骨架
5. 联调登录 → 线索提交 → 跟进 → 看板 全链路