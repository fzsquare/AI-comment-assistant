# AGENTS.md

本文档为 AI Agent / Codex 在本仓库工作提供指导。开发者主文档见 [README.md](./README.md)。

## 快速概览

本项目是 **PPK NFC 评价系统 MVP**：

- backend：Go + Gin + Gorm + JWT，对外提供 `/api/*`
- frontend：Vue 3 + Vite + Pinia + Axios，包含消费者落地页、商家后台、管理员后台
- database：MySQL，表结构以 `database/schema.sql` 和迁移 SQL 为准
- agent-service：Python + FastAPI + OpenAI Agents SDK，只供 backend 内部调用

核心链路：

```text
Browser
  -> frontend / public gateway
  -> Go backend (/api)
  -> MySQL
  -> agent-service (/generate-reviews)
```

浏览器不得直接访问 MySQL 或 agent-service。前端构建产物不得包含 `MYSQL_DSN`、`JWT_SECRET`、`AGENT_SERVICE_URL`、`AGENT_INTERNAL_TOKEN`、`LLM_API_KEY`。

## 常用 Workflow

### 阅读与修改

- 先读现有代码路径、配置和相邻实现，再修改。
- 优先使用 `rg` 搜索。
- 改动保持小而集中，不顺手重构无关模块。
- 工作区可能已有用户改动，不要回滚或清理未确认的改动。
- 手动编辑文件使用 `apply_patch`。

### 启动项目

完整本地部署优先使用脚本：

```bash
cp .env.deploy.example .env.deploy
scripts/deploy.sh start
```

脚本入口是 `http://127.0.0.1:8989`，会启动：

- `agent-service`：`127.0.0.1:8090`
- backend：`127.0.0.1:18989`
- public gateway / frontend：`0.0.0.0:8989`

`LLM_API_KEY` 为空时脚本默认停止；仅 UI/API 联调可设置 `ALLOW_EMPTY_LLM_KEY=true`，但 AI 生成会返回 503。

开发模式需要分别启动：

以下每段命令都从仓库根目录在单独终端执行。

```bash
# agent-service
cd agent-service
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python -m app.main
```

```bash
# backend
cd backend
APP_ENV=development \
APP_HOST=127.0.0.1 \
APP_PORT=8080 \
MYSQL_DSN="ppk_dev:ppk_dev_password@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET="dev-jwt-secret-change-me-32-bytes" \
ALLOWED_ORIGINS="http://127.0.0.1:5173,http://localhost:5173" \
AGENT_SERVICE_URL="http://127.0.0.1:8090" \
AGENT_INTERNAL_TOKEN="dev-agent-internal-token-change-me" \
go run ./cmd/server
```

```bash
# frontend
cd frontend
npm install
npm run dev -- --host 127.0.0.1 --port 5173
```

backend 和 agent-service 的 `AGENT_INTERNAL_TOKEN` 必须一致。真实 AI 生成还必须配置 `LLM_API_KEY`。

### 验证

按改动范围选择验证，跨模块改动要组合执行：

```bash
cd backend && go test ./...
cd agent-service && python3 tests/test_constraints.py
cd agent-service && python3 -m compileall app
cd frontend && npm run build
python3 scripts/check_frontend_flows.py --base-url http://127.0.0.1:8989
```

如果没有启动 `agent-service` 或没有配置 `LLM_API_KEY`，不要声称 AI 生成功能已端到端通过。

### 部署脚本

`scripts/deploy.sh` 支持：

```bash
scripts/deploy.sh start
scripts/deploy.sh restart
scripts/deploy.sh stop
scripts/deploy.sh status
scripts/deploy.sh logs
scripts/deploy.sh install
scripts/deploy.sh build
```

重要部署变量：

- `FRONTEND_PORT=8989`
- `BACKEND_PORT=18989`
- `AGENT_PORT=8090`
- `BASE_PATH=/ppk`
- `PUBLIC_ORIGIN=https://your-domain.com`
- `INIT_DB=true`
- `MIGRATE_DB=true`
- `LOAD_SEED=true`
- `ALLOW_EMPTY_LLM_KEY=true`
- `SMOKE_TEST=true`

## 架构地图

```text
backend/
  cmd/server/                 # Go 服务入口
  internal/config/            # 配置读取与生产校验
  internal/database/          # Gorm DB 初始化
  internal/handler/admin/     # 管理员 API
  internal/handler/merchant/  # 商家 API
  internal/handler/public/    # 消费者 API
  internal/middleware/        # CORS、JWT、角色校验
  internal/model/             # 数据模型
  internal/router/            # 依赖装配和路由注册
  internal/service/           # 业务逻辑、评价池、AI 调用

frontend/
  src/api/                    # Axios 请求封装
  src/router/                 # 路由、鉴权、document.title
  src/stores/                 # Pinia auth 状态
  src/views/landing/          # 消费者落地页
  src/views/merchant/         # 商家后台
  src/views/admin/            # 管理员后台

agent-service/
  app/main.py                 # FastAPI 入口，/health 和 /generate-reviews
  app/config.py               # LLM 与监听配置
  app/internal_auth.py        # X-Agent-Internal-Token 校验
  app/pipeline.py             # writer -> filter -> reviewer -> rewrite
  app/constraints/            # 平台、行业、人设、禁用词约束
  tests/test_constraints.py   # 约束测试

database/
  schema.sql                  # 全量建表
  seed.sql                    # 演示数据
  migrations/*.sql            # schema_migrations 去重执行的迁移
  migration-*.sql             # 旧库升级迁移

scripts/
  deploy.sh                   # 一键安装、构建、启动、停止
  serve_gateway.py            # 本地 public gateway
  check_frontend_flows.py     # 部署冒烟检查
```

## 关键规范

### 数据库

- backend 启动不 AutoMigrate。
- 全新环境先执行 `database/schema.sql`。
- 演示环境再执行 `database/seed.sql`。
- 旧库按需执行 `database/migrations/*.sql` 和 `database/migration-*.sql`。
- 修改模型时同步检查 SQL schema、迁移脚本、seed 数据和相关测试。

### backend

- 配置入口是 `backend/internal/config/config.go`。
- 路由装配入口是 `backend/internal/router/router.go`。
- HTTP 层按角色拆分：`admin`、`merchant`、`public`。
- 业务逻辑优先放在 `internal/service/`，不要把复杂规则堆到 handler。
- API 返回结构使用 `internal/pkg/response`。
- 生产环境不能使用弱 `JWT_SECRET`、空 `MYSQL_DSN`、空 `ALLOWED_ORIGINS` 或弱 `AGENT_INTERNAL_TOKEN`。

### frontend

- API 请求经 `frontend/src/api/http.ts` 和角色 API 文件封装。
- 路由与页面标题在 `frontend/src/router/index.ts` 统一维护。
- 不把服务端密钥、MySQL 地址、agent-service 地址写入任何 `VITE_*` 变量。
- `VITE_API_BASE_URL` 只能指向 Go backend 公开 API，例如 `/api`、`/ppk/api` 或 `https://api.example.com/api`。

### agent-service

- 默认只监听 `127.0.0.1:8090`。
- `POST /generate-reviews` 必须校验 `X-Agent-Internal-Token`。
- `LLM_API_KEY` 缺失时，健康检查可以通过，但生成必须失败。
- Go backend 只依赖 HTTP 契约；更换 LLM provider 时优先改 agent-service 内部。
- 不要将 `/generate-reviews` 暴露给浏览器、Nginx 公网入口或前端托管商。

### AI 生成与评价池

- 商家手动添加和 AI 生成评价都绑定到启用的平台入口。
- 消费者落地页先选择平台，再调用 `switch-review`。
- `switch-review` 必须带 `platformCode`。
- 评价池按 `store_id + platform_style` 隔离。
- 可发放库存是 `status='available' AND is_dispatched=0`。
- AI 生成任务失败时记录错误，不回退 mock 文案。
- Go backend 默认只保留 `grade` 为 `S/A/B` 的文案，阈值由 `AGENT_MIN_GRADE` 控制。

## 常见排障判断

- `connect: connection refused` 且目标是 `127.0.0.1:8090`：agent-service 没启动或端口不对。
- agent-service 返回 401：backend 与 agent-service 的 `AGENT_INTERNAL_TOKEN` 不一致。
- agent-service 返回 503：通常是 `LLM_API_KEY` 未配置。
- 页面能打开但接口失败：检查 Vite proxy、`VITE_API_BASE_URL`、backend 进程、`ALLOWED_ORIGINS`、Nginx/gateway。
- 登录失败或没有演示数据：检查是否导入 `database/seed.sql` 或设置 `LOAD_SEED=true`。
- `/ppk/` 子路径资源 404：检查 `BASE_PATH=/ppk`、`VITE_BASE_PATH=/ppk/`、`VITE_API_BASE_URL=/ppk/api` 和网关代理。

## 文档维护

- 开发者只需要读 [README.md](./README.md)。
- AI Agent 工作约束维护在本文档。
- 修改启动、端口、环境变量、安全边界、核心业务规则时，同步更新本文档和 README。
