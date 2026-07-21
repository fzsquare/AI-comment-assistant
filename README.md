# PPK NFC 评价系统 MVP

基于 **Gin + Vue 3 + MySQL + Python agent-service** 的 NFC 碰碰卡评价辅助系统 MVP。

消费者到店后碰 NFC 标签进入落地页，选择评价平台，获取推荐评价文案、图片素材，并跳转到目标平台发布。商家在后台维护门店、关键词、图片、平台入口和评价池；管理员维护商家、门店与 NFC 标签；AI 文案生成由内部 `agent-service` 提供。

面向 AI / Codex 的工作指南见 [AGENTS.md](./AGENTS.md)。开发、启动、部署、验证和排障说明都维护在本文。

## 快速概览

### 系统组成

```text
Browser
  -> frontend / public gateway
  -> Go backend public API (/api)
  -> local/private MySQL
  -> local/private agent-service
```

| 模块 | 技术栈 | 默认端口 | 说明 |
|---|---|---:|---|
| frontend | Vue 3 + Vite + Pinia + Axios | `5173` dev / `8989` gateway | 消费者落地页、商家后台、管理员后台 |
| backend | Gin + Gorm + JWT | `8080` dev / `18989` deploy | 对浏览器开放 `/api/*` |
| database | MySQL | `3306` | 使用 SQL 脚本建表，不在启动时 AutoMigrate |
| agent-service | FastAPI + OpenAI Agents SDK | `8090` | 仅供 backend 内部调用，不暴露给浏览器 |

生产或脚本部署时，只建议对公网开放 `8989` 或正式 Nginx/网关入口。MySQL、Go backend 内部端口和 agent-service 必须保持本机或私有网络访问。

### 目录结构

```text
.
├── backend/                    # Gin 后端服务
│   ├── cmd/server/             # 服务启动入口
│   └── internal/
│       ├── config/             # 环境变量读取与生产校验
│       ├── database/           # 数据库连接
│       ├── handler/            # admin / merchant / public HTTP 层
│       ├── middleware/         # CORS、JWT、角色校验
│       ├── model/              # Gorm 数据模型
│       ├── router/             # 路由注册与依赖装配
│       └── service/            # 认证、评价池、AI 生成调用
├── frontend/                   # Vue 3 前端
│   ├── src/api/                # Axios API 封装
│   ├── src/router/             # 路由与页面标题
│   ├── src/stores/             # Pinia 状态
│   └── src/views/              # landing / merchant / admin 页面
├── agent-service/              # 内部 Python 文案生成服务
│   ├── app/                    # FastAPI、LLM client、生成/评审 pipeline
│   └── tests/                  # 约束测试
├── database/                   # schema、seed、迁移脚本
├── scripts/                    # 一键部署、gateway、冒烟检查
├── .env.deploy.example         # 脚本部署环境模板
├── AGENTS.md                   # AI Agent / Codex 工作指南
└── README.md                   # 开发者主文档
```

## 当前实现范围

已实现：

- 商家端：登录、门店信息、关键词、图片、平台入口、评价列表、AI 生成任务。
- 管理员端：登录、商家列表、门店列表、NFC 标签创建/绑定/状态、生成任务、统计。
- 消费者端：落地页初始化、平台选择、推荐评价发放、换一换、事件上报、平台跳转。
- 核心逻辑：评价池按 `store_id + platform_style` 隔离发放；发放后设置 `is_dispatched=1`；评价池不足时触发补货；AI 文案默认只保留 B 级及以上。
- 安全边界：浏览器只访问 frontend 和 Go backend `/api`；不直接访问 MySQL 或 agent-service；LLM key 不进入前端构建产物。

尚未完善：

- 对象存储上传、生产级日志/指标/告警、LLM 成本监控、审核流、更细粒度权限、完整统计后台、浏览器自动化 UI 回归。

## 环境要求

- Go `1.26+`，以 [backend/go.mod](./backend/go.mod) 为准
- Node.js `18+`
- npm `9+`
- Python `3.10+`
- MySQL `8.0+`

macOS 本地开发可以直接运行。Windows / WSL 环境建议在 Linux 路径下执行 `npm install` 和 `npm run dev`，避免前端依赖在 UNC 路径下触发平台二进制兼容问题。

## 推荐启动方式

### 一键本地部署模式

适合完整看效果、模拟单机生产拓扑。入口是 `http://127.0.0.1:8989`。

```bash
cp .env.deploy.example .env.deploy
# 编辑 .env.deploy，至少配置 MYSQL_DSN 或 DB_APP_PASSWORD。
# 需要真实 AI 生成时必须配置 LLM_API_KEY。
scripts/deploy.sh start
```

常用命令：

```bash
scripts/deploy.sh status
scripts/deploy.sh logs
scripts/deploy.sh restart
scripts/deploy.sh stop
```

脚本会执行：

- 读取 `.env.deploy`，必要时生成 `JWT_SECRET` 和 `AGENT_INTERNAL_TOKEN` 到 `.deploy/runtime.env`
- 安装 Go / npm / Python 项目依赖
- 构建 frontend 和 backend
- 启动 `agent-service`：`127.0.0.1:8090`
- 启动 backend：`127.0.0.1:18989`
- 启动 public gateway：`0.0.0.0:8989`
- 通过 `scripts/check_frontend_flows.py` 做基本冒烟检查

如果没有 LLM key，只想先启动 UI/API，可在 `.env.deploy` 中设置：

```bash
ALLOW_EMPTY_LLM_KEY=true
```

这种情况下 `agent-service` 可以启动和健康检查，但 AI 生成会返回 503。

新环境需要初始化数据库时，可在 `.env.deploy` 中设置：

```bash
INIT_DB=true
MIGRATE_DB=true
LOAD_SEED=true
MYSQL_ROOT_USER=root
MYSQL_ROOT_PASSWORD=<root-password>
DB_APP_PASSWORD=<strong-db-password>
```

`LOAD_SEED=true` 会导入演示账号和示例门店数据。
`MIGRATE_DB=true` 会按 `schema_migrations` 去重执行 `database/migrations/*.sql`，适合旧库升级；`INIT_DB=true` 时迁移会在 schema 之后、seed 之前执行。

### 本地开发模式

适合改代码。入口是 `http://127.0.0.1:5173`，需要分别启动 MySQL、agent-service、backend、frontend。

1. 初始化数据库

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p -e "CREATE DATABASE IF NOT EXISTS ppk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -h 127.0.0.1 -P 3306 -u root -p -e "CREATE USER IF NOT EXISTS 'ppk_dev'@'127.0.0.1' IDENTIFIED BY 'ppk_dev_password'; GRANT SELECT, INSERT, UPDATE, DELETE ON ppk.* TO 'ppk_dev'@'127.0.0.1'; FLUSH PRIVILEGES;"
mysql -h 127.0.0.1 -P 3306 -u root -p ppk < database/schema.sql
mysql -h 127.0.0.1 -P 3306 -u root -p ppk < database/seed.sql
```

已有旧库升级时，按需执行：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p ppk < database/migrations/0001_store_types_uuid.sql
mysql -h 127.0.0.1 -P 3306 -u root -p ppk < database/migration-2026-platform-review-pool.sql
mysql -h 127.0.0.1 -P 3306 -u root -p ppk < database/migration-2026-add-review-tags.sql
mysql -h 127.0.0.1 -P 3306 -u root -p ppk < database/migration-2026-review-feedback.sql
```

2. 启动 agent-service

```bash
cd agent-service
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
cp .env.example .env
```

编辑 `agent-service/.env`：

```bash
LLM_API_KEY=<your-llm-api-key>
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-5.4
AGENT_HOST=127.0.0.1
AGENT_PORT=8090
AGENT_INTERNAL_TOKEN=dev-agent-internal-token-change-me
```

启动并检查：

```bash
python -m app.main
curl http://127.0.0.1:8090/health
```

3. 启动 backend

```bash
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

4. 启动 frontend

```bash
cd frontend
npm install
cp .env.example .env.local
npm run dev -- --host 127.0.0.1 --port 5173
```

前端开发环境默认通过 Vite proxy 把 `/api` 转发到 `VITE_DEV_API_PROXY_TARGET`，模板默认值是 `http://127.0.0.1:8080`。

## 演示账号

只有导入 [database/seed.sql](./database/seed.sql) 或脚本部署时设置 `LOAD_SEED=true` 后才存在：

| 角色 | 账号 | 密码 | 入口 |
|---|---|---|---|
| 管理员 | `admin` | `123456` | `/admin/login` |
| 商家 | `merchant` | `123456` | `/merchant/login` |

演示消费者落地页：

- 开发模式：`http://127.0.0.1:5173/landing/11111111-1111-4111-8111-111111111111`
- 部署模式：`http://127.0.0.1:8989/landing/11111111-1111-4111-8111-111111111111`
- `BASE_PATH=/ppk` 时：`http://127.0.0.1:8989/ppk/landing/11111111-1111-4111-8111-111111111111`

这里的 UUID 是 seed 数据里的示例门店标识，不是最终面向消费者的短链格式。

## 配置说明

### backend 环境变量

| 变量 | 说明 | 本地默认 |
|---|---|---|
| `APP_ENV` | 运行环境，生产用 `production` | `development` |
| `APP_HOST` | backend 监听地址 | `127.0.0.1` |
| `APP_PORT` | backend 监听端口 | `8080` |
| `MYSQL_DSN` | MySQL 连接串 | 本地 `ppk_dev` DSN |
| `JWT_SECRET` | JWT 密钥，生产至少 32 字符 | 本地开发密钥 |
| `ALLOWED_ORIGINS` | 允许访问 API 的浏览器 origin，不能用 `*` | 本地 5173 origin |
| `AGENT_SERVICE_URL` | backend 内部调用 agent-service 地址 | `http://127.0.0.1:8090` |
| `AGENT_INTERNAL_TOKEN` | backend 与 agent-service 共享令牌 | 本地开发令牌 |
| `AGENT_MIN_GRADE` | AI 评价入池最低等级 | `B` |
| `MAX_REVIEW_GENERATE_COUNT` | 单次生成数量上限 | `50` |
| `DEFAULT_REVIEW_TARGET_COUNT` | 默认生成目标数量 | `10` |
| `UPLOAD_DIR` | 商家上传图片保存目录 | `./uploads` |
| `PUBLIC_BASE_URL` | 生成绝对落地页/图片 URL 的前缀 | 空 |
| `PUBLIC_BASE_PATH` | 子路径部署前缀，例如 `/ppk` | 空 |

`APP_ENV=production` 时，backend 会拒绝弱 `JWT_SECRET`、空 `MYSQL_DSN`、空 `ALLOWED_ORIGINS` 或弱 `AGENT_INTERNAL_TOKEN`。

### frontend 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `VITE_API_BASE_URL` | 浏览器可访问的 Go API 入口 | `/api` |
| `VITE_BASE_PATH` | Vite base，子路径部署如 `/ppk/` | `/` |
| `VITE_DEV_API_PROXY_TARGET` | dev server 的 `/api` 代理目标 | `http://127.0.0.1:8080` |

前端 `.env*` 只能包含浏览器可见变量。不要写入 `MYSQL_DSN`、`AGENT_SERVICE_URL`、`AGENT_INTERNAL_TOKEN`、`JWT_SECRET`、`LLM_API_KEY`。

### agent-service 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `LLM_API_KEY` | LLM 供应商 API key | 空 |
| `LLM_BASE_URL` | OpenAI 兼容端点 | `https://api.openai.com/v1` |
| `LLM_MODEL` | 文案生成模型 | `gpt-5.4` |
| `MIN_PASS_SCORE` | 评审通过分数 | `80` |
| `MAX_REVISE_ROUNDS` | 不达标重写轮数 | `2` |
| `MAX_CONCURRENCY` | 批量生成并发 | `5` |
| `AGENT_HOST` | 监听地址 | `127.0.0.1` |
| `AGENT_PORT` | 监听端口 | `8090` |
| `AGENT_INTERNAL_TOKEN` | 与 backend 一致的内部令牌 | 无 |

## 核心业务规则

### 评价池

- 可发放库存：`status='available' AND is_dispatched=0`
- 发放后设置 `is_dispatched=1`
- 评价按 `store_id + platform_style` 隔离，消费者选择平台后只领取该平台文案
- 商家手工添加评价和 AI 生成评价都必须绑定到已启用的平台入口
- AI 生成不再回退内置 mock；agent-service 不可用时生成任务失败并记录错误

### 平台入口

- 商家配置平台入口，例如美团、大众点评、小红书、抖音
- 消费者打开 NFC 落地页后先选择平台，再请求 `switch-review`
- `switch-review` 请求体必须带 `platformCode`
- 页面只展示所选平台的发布按钮

### 数据库

- backend 启动不自动建表
- 全新环境执行 `database/schema.sql`
- 演示环境再执行 `database/seed.sql`
- 旧库按实际版本补充执行 `database/migrations/*.sql` 和 `database/migration-*.sql`

## 主要接口

### 商家端

- `POST /api/merchant/auth/login`
- `GET /api/merchant/store/detail`
- `PUT /api/merchant/store/detail`
- `GET|POST|PUT|DELETE /api/merchant/store/keywords`
- `GET|POST|DELETE /api/merchant/store/images`
- `GET|POST|PUT|DELETE /api/merchant/store/platform-links`
- `GET|POST|PUT|DELETE /api/merchant/reviews`
- `POST /api/merchant/reviews/generate`
- `GET /api/merchant/review-generation-tasks`

### 管理员端

- `POST /api/admin/auth/login`
- `GET /api/admin/merchants`
- `PUT /api/admin/merchants/:id/status`
- `GET /api/admin/stores`
- `PUT /api/admin/stores/:id/status`
- `GET|POST /api/admin/nfc-tags`
- `PUT /api/admin/nfc-tags/:id/bind`
- `PUT /api/admin/nfc-tags/:id/status`
- `GET /api/admin/review-generation-tasks`
- `GET /api/admin/stats`

### 消费者端

- `GET /api/public/landing/:token/init`
- `POST /api/public/landing/:token/switch-review`
- `POST /api/public/landing/:token/events`

## 部署

### 单机脚本部署

推荐 MVP 演示环境直接使用：

```bash
cp .env.deploy.example .env.deploy
scripts/deploy.sh start
```

拓扑：

```text
Browser
   -> public gateway :8989
      -> frontend/dist
      -> /api -> backend 127.0.0.1:18989
         -> MySQL 127.0.0.1:3306 or private DB
         -> agent-service 127.0.0.1:8090
```

如果公网入口由 Nginx 的 `/ppk/` 统一反代到本机 `8989`，设置：

```bash
BASE_PATH=/ppk
PUBLIC_ORIGIN=https://your-domain.com
```

脚本会让前端资源、API、上传资源和落地页链接统一走 `/ppk/...`。

已有数据库在执行 `database/migrations/0007_publish_stats_index.sql` 前，如果
`review_display_logs` 已有历史数据，必须先确认旧 backend 主机写入无时区
`DATETIME` 时采用的时区，并按需修正历史记录。审计完成后再在部署配置中设置：

```bash
HISTORICAL_DATETIME_TIMEZONE_AUDITED=true
```

脚本生成的生产 MySQL DSN 使用 `loc=Asia%2FShanghai`；不要把开发机的
`loc=Local` 复制为生产数据库时区配置。

### 手动生产部署

手动部署时保持同样边界：

1. MySQL 执行 `database/schema.sql`，演示环境执行 `database/seed.sql`
2. `agent-service` 绑定 `127.0.0.1:8090` 或内网地址，配置 `LLM_API_KEY` 和 `AGENT_INTERNAL_TOKEN`
3. backend 绑定 `127.0.0.1:18989`，配置生产 `MYSQL_DSN`、`JWT_SECRET`、`ALLOWED_ORIGINS`、`AGENT_SERVICE_URL`、`AGENT_INTERNAL_TOKEN`
4. frontend 执行 `npm ci && npm run build`
5. Nginx 或网关只把公开流量转给 frontend 和 backend `/api`

根路径 Nginx 示例：

```nginx
server {
    listen 80;
    server_name app.example.com;

    root /opt/ppk/frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:18989;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

不要通过 Nginx 或 API 网关暴露 `/generate-reviews`。

## 验证

### 后端

```bash
cd backend
go test ./...
```

### agent-service

```bash
cd agent-service
python3 tests/test_constraints.py
python3 -m compileall app
curl http://127.0.0.1:8090/health
```

`/health` 只证明服务进程可用。真实生成还需要有效 `LLM_API_KEY`。

### 前端

```bash
cd frontend
npm run build
```

### 部署冒烟

```bash
python3 scripts/check_frontend_flows.py --base-url http://127.0.0.1:8989
```

如果设置了 `BASE_PATH=/ppk`：

```bash
python3 scripts/check_frontend_flows.py --base-url http://127.0.0.1:8989/ppk
```

## 常见问题

### AI 生成报 `connect: connection refused`

backend 正在访问 `AGENT_SERVICE_URL`，但 `127.0.0.1:8090` 没有进程监听。启动 `agent-service`，并确认 backend 与 agent-service 的 `AGENT_INTERNAL_TOKEN` 一致。

### AI 生成返回 401

`AGENT_INTERNAL_TOKEN` 不一致，或请求没有携带 `X-Agent-Internal-Token`。检查 backend 和 agent-service 的环境变量。

### AI 生成返回 503

通常是 `LLM_API_KEY` 为空。`agent-service` 可以健康检查通过，但真实生成必须配置可用 LLM key。

### 页面能打开但接口失败

检查：

- backend 是否启动
- Vite proxy 或 `VITE_API_BASE_URL` 是否指向 Go backend
- `ALLOWED_ORIGINS` 是否包含当前前端 origin
- Nginx / gateway 是否正确代理 `/api`
- 是否直接访问了内部端口而不是公开入口

### 登录失败或没有演示门店

确认已执行 `database/seed.sql`，或脚本部署时设置了 `LOAD_SEED=true`。

### 子路径 `/ppk/` 部署资源 404

确认构建时 `VITE_BASE_PATH=/ppk/`，API 为 `/ppk/api`，并且 Nginx 或 gateway 把 `/ppk/` 转给同一个入口。

## 上线前检查

- [ ] `go test ./...` 通过
- [ ] `python3 tests/test_constraints.py` 通过
- [ ] `npm run build` 通过
- [ ] `APP_ENV=production`
- [ ] `JWT_SECRET`、`AGENT_INTERNAL_TOKEN` 已替换为强随机值
- [ ] `MYSQL_DSN` 使用最小权限账号
- [ ] `ALLOWED_ORIGINS` 只包含真实前端域名
- [ ] `LLM_API_KEY` 只存在于服务器端
- [ ] 前端构建产物不包含 MySQL、agent-service、JWT、LLM 等服务端配置
- [ ] 公网不开放 MySQL、backend 内部端口、agent-service
- [ ] 商家、管理员、消费者主流程完成端到端验证

## 文档导航

- [AGENTS.md](./AGENTS.md)：AI Agent 在本仓库工作的规则、架构速查、验证要求
- [README.md](./README.md)：开发、启动、部署、验证、排障主文档
- [agent-service/README.md](./agent-service/README.md)：文案生成服务内部说明
