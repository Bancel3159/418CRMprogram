# 418 CRM 数据中台

企业级 CRM / 数据中台项目。三个端协同：

- **Web 端**：主管及以上使用，全量数据看板、线索管理、报表分析
- **桌面端 A（lead）**：一线员工提交线索
- **桌面端 B（follow）**：跟进人员维护客户

## 技术栈

| 端 | 技术 |
|----|------|
| Web 前端 | TypeScript + Vue 3 + Vite + Naive UI + Pinia |
| 桌面端 | Tauri (Rust) + Vue 3 + Naive UI |
| 后端 | Go 1.22+ + Gin + GORM + PostgreSQL + Redis |
| 部署 | Docker / Docker Compose |

## 目录结构

```
.
├── backend/         Go 后端
├── web/             Web 端（主管后台）
├── desktop-lead/    桌面端 A：提交线索
├── desktop-follow/  桌面端 B：跟进维护
├── shared/          三端共享类型
└── docs/            项目文档
```

详细架构见 [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)。

## 快速开始

### 后端

```bash
cd backend
cp .env.example .env   # 配置数据库、Redis 等
go mod tidy
go run cmd/server/main.go
```

### Web 端

```bash
cd web
pnpm install
pnpm dev
```

### 桌面端

```bash
cd desktop-lead   # 或 desktop-follow
pnpm install
pnpm tauri dev
```

## 文档

- [架构设计](./docs/ARCHITECTURE.md)
- [接口规范](./docs/API.md)
- [数据库设计](./docs/DATABASE.md)
- [部署文档](./docs/DEPLOY.md)