# PPK NFC 评价系统 MVP

基于 **Gin + Vue 3 + MySQL** 的 NFC 碰碰卡评价辅助系统 MVP。

本项目面向线下商家：消费者到店后通过手机碰一碰 NFC 标签，打开落地页，获取推荐评价文案、图片素材，并跳转到目标平台进行评论或发布内容。

当前仓库包含：

- 后端服务：Gin + Gorm + JWT
- 内部 AI agent-service：Python 服务，仅供后端在服务器本机或私有网络调用
- 前端应用：Vue 3 + Vite + Pinia + Axios
- 数据库脚本：MySQL schema / seed
- 部署与运维文档：`README.md`、`README-DEPLOY.md`、`agent-service/README.md`

云端访问边界：

```text
Browser
  -> Frontend provider / static hosting
  -> Go backend public API (/api)
  -> local/private MySQL
  -> local/private agent-service
```

浏览器只访问前端静态资源与 Go backend 的公开 API。根路径部署时 API 为 `/api`；如果 Nginx 把 `/ppk/` 统一反代到 `8989`，前端会使用 `/ppk/api`，资源会使用 `/ppk/assets` 与 `/ppk/uploads`。数据库和 agent-service 不开放公网，也不配置到前端环境变量中；前端只允许使用 `VITE_API_BASE_URL` 指向 Go backend API。

---

## 1. 项目目录结构

```text
ppk/
├── backend/                    # Gin 后端服务
│   ├── cmd/server/             # 服务启动入口
│   ├── internal/
│   │   ├── config/             # 配置读取
│   │   ├── database/           # 数据库连接初始化
│   │   ├── handler/            # HTTP 接口层
│   │   │   ├── admin/          # 管理员端接口
│   │   │   ├── merchant/       # 商家端接口
│   │   │   └── public/         # 消费者公开接口
│   │   ├── middleware/         # CORS、JWT 鉴权中间件
│   │   ├── model/              # Gorm 数据模型
│   │   ├── pkg/                # 通用工具
│   │   │   ├── auth/           # JWT / bcrypt
│   │   │   ├── response/       # 统一响应结构
│   │   │   └── utils/          # 随机串等工具
│   │   ├── router/             # 路由注册
│   │   └── service/            # 业务逻辑层
│   ├── go.mod
│   └── go.sum
├── frontend/                   # Vue 前端项目
│   ├── src/
│   │   ├── api/                # Axios API 封装
│   │   ├── router/             # 前端路由
│   │   ├── stores/             # Pinia 状态
│   │   ├── styles/             # 全局样式
│   │   ├── views/
│   │   │   ├── admin/          # 管理员页面
│   │   │   ├── landing/        # 消费者落地页
│   │   │   └── merchant/       # 商家后台页面
│   │   ├── App.vue
│   │   └── main.ts
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── database/                   # MySQL 初始化脚本
│   ├── schema.sql
│   ├── seed.sql
│   └── migration-2026-platform-review-pool.sql
├── agent-service/              # 内部 Python agent 服务，不直接暴露给浏览器
└── README.md
```

---

## 2. 模块划分说明

## 2.1 backend 模块

### `cmd/server`
后端启动入口，负责：
- 读取配置
- 初始化数据库连接
- 注册路由
- 启动 Gin HTTP 服务

### `internal/config`
读取运行配置，当前主要支持：
- `APP_PORT`
- `APP_ENV`
- `MYSQL_DSN`
- `JWT_SECRET`
- `ALLOWED_ORIGINS`
- `AGENT_SERVICE_URL`
- `AGENT_INTERNAL_TOKEN`
- `AGENT_MIN_GRADE`
- `MAX_REVIEW_GENERATE_COUNT`
- `DEFAULT_REVIEW_TARGET_COUNT`

### `internal/database`
负责数据库连接初始化。

当前采用 **方案 A**：
- 只建立数据库连接
- 不在服务启动时自动执行 `AutoMigrate`
- 表结构以 `database/schema.sql` 为准

### `internal/model`
定义核心数据模型，包含：
- `AdminUser`
- `MerchantUser`
- `Store`
- `StoreKeyword`
- `StoreImage`
- `StorePlatformLink`
- `ReviewItem`
- `ReviewDisplayLog`
- `ReviewGenerationTask`
- `NFCTag`

### `internal/service`
核心业务逻辑层：

- `auth_service.go`
  - 商家登录
  - 管理员登录
  - JWT 生成

- `review_pool_service.go`
  - 消费者初始化评价发放
  - “换一换”发放
  - 可发放库存检查
  - 自动补货逻辑

- `review_generator_agent.go`
  - 主生成器，通过 HTTP 调用内部 Python `agent-service`
  - 请求 `/generate-reviews` 时携带 `X-Agent-Internal-Token`
  - 默认只将 B 级及以上文案写入评价池

- `review_generator_mock.go`
  - 仅作为兜底生成器
  - 当评价池为空且 agent-service 不可用时即时补 1 条，避免落地页白屏

### `internal/handler`
HTTP 接口层，按角色拆分：

- `merchant/`：商家后台接口
- `admin/`：管理员后台接口
- `public/`：消费者公开接口

### `internal/middleware`
中间件：
- CORS
- JWT 鉴权
- 角色限制

### `internal/pkg`
通用基础能力：
- `auth/`：密码哈希、Token 解析
- `response/`：统一返回结构
- `utils/`：随机字符串等工具

---

## 2.2 frontend 模块

### `src/api`
前端 API 封装，按角色划分：
- `public.ts`
- `merchant.ts`
- `admin.ts`
- `http.ts`：Axios 实例与 token 注入

### `src/router`
前端路由定义，当前主要页面：
- `/landing/:token`
- `/merchant/login`
- `/merchant/console`
- `/admin/login`
- `/admin/console`

### `src/stores`
Pinia 状态管理，当前主要是：
- 登录 token
- 用户角色

### `src/views`
页面按角色拆分：

- `landing/`
  - 消费者落地页
  - 先选择评价平台，再按平台展示文案、复制、换一换、平台入口跳转

- `merchant/`
  - 商家登录页
  - 商家控制台
  - 门店、关键词、图片、平台入口、评价、生成任务展示

- `admin/`
  - 管理员登录页
  - 管理员控制台
  - 商家、门店、NFC 标签、任务、统计展示

---

## 2.3 database 模块

### `schema.sql`
数据库结构脚本，负责创建：
- 用户表
- 门店表
- 关键词表
- 图片表
- 平台链接表
- 评价表
- 评价日志表
- 生成任务表
- NFC 标签表

### `seed.sql`
演示数据脚本，负责插入：
- 管理员账号
- 商家账号
- 示例门店
- 关键词
- 图片
- 平台入口
- NFC 标签
- 初始评价数据
  - 默认按 `meituan`、`dianping` 两个平台写入评价池，便于演示平台隔离发放

---

## 2.4 agent-service 模块

内部 Python AI 文案生成服务，默认监听 `127.0.0.1:8090`，只允许 Go backend 在服务器本机或私有网络调用。

关键文件：

- `app/main.py`
  - FastAPI 入口
  - `GET /health` 本机探活
  - `POST /generate-reviews` 生成多平台评价文案
  - 校验 `X-Agent-Internal-Token`

- `app/pipeline.py`
  - 选择平台 writer agent
  - 生成 JSON 文案
  - 禁用词硬过滤
  - reviewer agent 打分
  - 不达标时重写

- `app/client.py`
  - 使用 OpenAI Agents SDK
  - 通过 `OpenAIChatCompletionsModel` 调用任意 OpenAI 兼容端点

- `app/config.py`
  - 读取 `LLM_API_KEY`、`LLM_BASE_URL`、`LLM_MODEL`
  - 读取 `AGENT_HOST`、`AGENT_PORT`、`AGENT_INTERNAL_TOKEN`

浏览器和前端构建产物不应包含 agent-service 地址或任何 LLM key。

---

## 3. 当前实现范围（MVP）

当前实现的是可运行 MVP，不是完整 V1。

### 已实现

#### 商家端
- 登录
- 查看/编辑门店信息
- 关键词 CRUD
- 图片列表 / 新增 / 删除
- 平台链接列表 / 新增 / 启停 / 删除
- 评价列表 / 新增 / 删除
- 手动触发生成
- 查看生成任务

#### 管理员端
- 登录
- 查看商家列表
- 查看门店列表
- 创建 NFC 标签
- 绑定 NFC 标签
- 修改标签状态
- 查看生成任务
- 查看统计信息

#### 消费者端
- 落地页初始化
- 获取推荐评价
- “换一换”
- 事件上报
- 平台入口展示
- 图片展示

#### 核心领域逻辑
- 可发放库存定义：`status=available AND is_dispatched=0`
- 低于阈值自动补货
- Python agent-service 作为主 AI 文案生成器
- Go backend 调用 agent-service 时使用内部 token
- 默认只将 B 级及以上 AI 文案写入评价池
- mock 生成器仅保留为空池兜底，不作为主生成路径
- JWT 鉴权
- CORS 白名单
- 生产环境关键配置校验

### 尚未完善
- 对象存储上传
- AI 生成成本、限流、监控与告警
- agent-service 进程守护、日志落盘与失败重试策略
- 更细粒度权限控制
- 审核流
- 完整统计后台
- 前端复杂交互与样式打磨
- 浏览器自动化 UI 验证

---

## 4. 关键业务规则

### 4.1 评价池
- 默认目标库存：50
- 自动补货阈值：可发放库存 `< 20`
- 自动补货目标：补足到 50
- 发放后设置 `is_dispatched = true`
- 评价池按 `store_id + platform_style` 隔离发放；消费者选择平台后，只会领取该平台的可用评价
- 商家手工添加评价和 AI 生成评价都必须绑定到一个已启用的平台入口

### 4.2 平台入口
- 平台入口由商家配置
- 消费者打开 NFC 落地页后先选择平台；选择后后端才通过 `platformCode` 发放对应平台的评价文案
- 消费者点击发布按钮后跳转到所选平台，页面不会同时展示多个平台跳转按钮
- 事件会记录到 `review_display_logs`

### 4.3 数据库初始化策略
当前项目启动 **不自动建表**。

必须先执行：
- `database/schema.sql`
- `database/seed.sql`

然后再启动 backend。

---

## 5. 环境要求

建议环境：

- Go：`1.26+`（需满足 `backend/go.mod` 中的 `go 1.26`）
- Node.js：建议 `18+`
- npm：建议 `9+`
- MySQL：`8.0+`
- 建议在 **WSL/Linux 路径** 下运行 frontend / backend 命令

> 注意：当前环境下，frontend 如果直接走 Windows UNC 路径执行，可能遇到 `esbuild` 路径兼容问题。建议统一在 WSL 内执行。

---

## 6. 配置说明

## 6.1 backend 环境变量

可通过环境变量覆盖默认配置：

| 变量名 | 说明 | 默认值 |
|---|---|---|
| `APP_HOST` | 后端监听地址，生产建议只绑定本机 | `127.0.0.1` |
| `APP_PORT` | 后端监听端口 | `8080` |
| `APP_ENV` | 运行环境，生产使用 `production` | `development` |
| `MYSQL_DSN` | MySQL 连接串，生产使用最小权限账号 | 本地开发 DSN |
| `JWT_SECRET` | JWT 密钥，生产至少 32 字符 | 本地开发密钥 |
| `ALLOWED_ORIGINS` | 允许访问 API 的前端 origin | 本地开发 origin |
| `AGENT_SERVICE_URL` | backend 内部调用 agent-service 的地址 | `http://127.0.0.1:8090` |
| `AGENT_INTERNAL_TOKEN` | backend 与 agent-service 共享的内部令牌 | 本地开发令牌 |

建议启动时显式传入：

```bash
APP_ENV=production
APP_HOST=127.0.0.1
APP_PORT=18989
MYSQL_DSN="ppk_app:<password>@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"
JWT_SECRET="<random-32-plus-char-secret>"
ALLOWED_ORIGINS="https://app.example.com"
AGENT_SERVICE_URL="http://127.0.0.1:8090"
AGENT_INTERNAL_TOKEN="<random-32-plus-char-token>"
```

## 6.2 frontend 环境变量

前端只允许配置浏览器可见的 API 入口：

```bash
cd frontend
cp .env.example .env.local
```

`frontend/.env.example` 内容：

```bash
VITE_API_BASE_URL=/api
VITE_BASE_PATH=/
VITE_DEV_API_PROXY_TARGET=http://127.0.0.1:8080
```

开发环境默认通过 Vite proxy 把 `/api` 转发到 `VITE_DEV_API_PROXY_TARGET`。生产根路径构建未配置 `VITE_API_BASE_URL` 时，Axios 默认使用同源 `/api`；子路径构建设置 `VITE_BASE_PATH=/ppk/` 后，Axios 默认使用 `/ppk/api`。如果前端静态托管与 API 不同域，只能把它设置为 Go backend 的公开 API，例如 `https://api.example.com/api`。不要在前端 `.env` 中配置 MySQL、agent-service、JWT 密钥或任何服务端 secret。

---

## 7. 部署指南

## 7.1 云端访问边界

生产环境建议采用以下链路：

```text
Browser
  -> public gateway :8989
  -> local Go backend 127.0.0.1:18989 (/api)
  -> local/private MySQL
  -> local/private agent-service
```

默认部署脚本会在 `8989` 上提供前端静态资源，并把公开 API 反向代理到本机 backend `127.0.0.1:18989`。根路径部署时公开 API 是 `/api`；设置 `BASE_PATH=/ppk` 时公开 API 是 `/ppk/api`。客户资源信息、门店数据、评价池与生成任务都通过 Go backend 鉴权后读取；浏览器不直接访问 MySQL 或 agent-service。

脚本部署时生产环境只应对公网开放：

- `8989`：前端页面和同源 API（根路径 `/api`，或 `BASE_PATH=/ppk` 时 `/ppk/api`）

必须保持本机或私有网络访问：

- MySQL：只允许 backend 使用 `MYSQL_DSN` 连接
- Go backend：默认监听 `127.0.0.1:18989`
- agent-service：默认监听 `127.0.0.1:8090`，只允许 backend 使用 `AGENT_INTERNAL_TOKEN` 调用
- `MYSQL_DSN`、`JWT_SECRET`、`AGENT_SERVICE_URL`、`AGENT_INTERNAL_TOKEN`、`LLM_API_KEY` 等服务端变量

## 7.2 一键脚本部署

推荐云服务器单机部署使用：

```bash
cp .env.deploy.example .env.deploy
# 编辑 .env.deploy，至少确认 MYSQL_DSN 或 DB_APP_PASSWORD、LLM_API_KEY 已填写
scripts/deploy.sh start
```

新服务器如果希望直接使用默认管理员/商家账号，需要在 `.env.deploy` 中同时设置 `INIT_DB=true` 与 `LOAD_SEED=true`；否则脚本只建表，不会导入 `admin / 123456` 和 `merchant / 123456`。

脚本会完成：

- 安装 Go / npm / Python 项目依赖
- 构建 frontend，并按 `BASE_PATH` 使用同源 API
- 构建 backend
- 启动 agent-service：`127.0.0.1:8090`
- 启动 backend：`127.0.0.1:18989`
- 启动 public gateway：`0.0.0.0:8989`
- 通过 public gateway 自检前端页面、同源 API 代理与后台管理接口

常用命令：

```bash
scripts/deploy.sh status
scripts/deploy.sh logs
scripts/deploy.sh restart
scripts/deploy.sh stop
```

如果本机 MySQL 已创建并导入 schema，只需要在 `.env.deploy` 中设置 `MYSQL_DSN`。如果希望脚本尝试初始化 MySQL，可设置：

```bash
INIT_DB=true
LOAD_SEED=true
DB_APP_PASSWORD=<strong-db-password>
# 如果 MySQL 看到的 backend 来源不是 127.0.0.1，按实际来源调整
DB_APP_HOST=127.0.0.1
MYSQL_ROOT_USER=root
MYSQL_ROOT_PASSWORD=<root-password>
```

`JWT_SECRET` 与 `AGENT_INTERNAL_TOKEN` 不填时，脚本会自动生成并保存到 `.deploy/runtime.env`。`.env.deploy` 与 `.deploy/` 已被 git 忽略，不应提交。

脚本启动前会先检查 MySQL host/port 是否能从服务器连通，并默认要求 `LLM_API_KEY` 不为空。启动后会运行 `scripts/check_frontend_flows.py`，从 `http://127.0.0.1:8989${BASE_PATH}` 检查 SPA 页面、同源 API 代理、管理员与商家后台接口；`LOAD_SEED=false` 时会跳过默认账号登录检查。若只想先启动 UI/API 而暂不启用 AI 生成，可在 `.env.deploy` 中设置 `ALLOW_EMPTY_LLM_KEY=true`。

如果公网 Nginx 只反代 `/ppk/` 到本机 `8989`，在 `.env.deploy` 中设置：

```bash
BASE_PATH=/ppk
PUBLIC_ORIGIN=https://your-domain.com
```

此时写入 NFC 的链接会是 `https://your-domain.com/ppk/landing/<store-uuid>`，前端资源和接口也都会走 `/ppk/...`。

## 7.3 手动单机部署流程

适合需要接入 Nginx / systemd 的环境。公网入口仍建议只暴露统一入口，backend、MySQL、agent-service 都绑定在本机或内网。

### 1. 初始化 MySQL

```bash
mysql -u root -p -e "CREATE DATABASE IF NOT EXISTS ppk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p -e "CREATE USER IF NOT EXISTS 'ppk_app'@'127.0.0.1' IDENTIFIED BY '<strong-password>'; GRANT SELECT, INSERT, UPDATE, DELETE ON ppk.* TO 'ppk_app'@'127.0.0.1'; FLUSH PRIVILEGES;"
mysql -u root -p ppk < database/schema.sql

# 仅演示环境导入
mysql -u root -p ppk < database/seed.sql
```

### 2. 启动 agent-service

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
AGENT_HOST=127.0.0.1
AGENT_PORT=8090
AGENT_INTERNAL_TOKEN=<random-32-plus-char-token>
```

启动服务并检查：

```bash
python -m app.main
curl http://127.0.0.1:8090/health
```

### 3. 启动 backend

```bash
cd backend
go build -o ppk-server ./cmd/server

APP_ENV=production \
APP_HOST=127.0.0.1 \
APP_PORT=18989 \
MYSQL_DSN="ppk_app:<strong-password>@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET="<random-32-plus-char-secret>" \
ALLOWED_ORIGINS="https://app.example.com" \
AGENT_SERVICE_URL="http://127.0.0.1:8090" \
AGENT_INTERNAL_TOKEN="<same-token-as-agent-service>" \
./ppk-server
```

### 4. 构建 frontend

```bash
cd frontend
npm ci
VITE_API_BASE_URL=/api npm run build
```

Nginx `/ppk/` 子路径部署时：

```bash
VITE_BASE_PATH=/ppk/ VITE_API_BASE_URL=/ppk/api npm run build
```

同域反向代理部署时可以不配置 `VITE_API_BASE_URL`，前端默认请求与 `VITE_BASE_PATH` 对应的同源 API。根路径是 `/api`，`/ppk/` 子路径是 `/ppk/api`。如果前端托管商和 API 域名不同，只在前端构建环境配置：

```bash
VITE_API_BASE_URL=https://api.example.com/api
```

生产构建前确认没有把 `localhost`、MySQL、agent-service 或任何服务端 secret 写入前端 `.env.production` / `.env.local`。

### 5. Nginx 同域反向代理示例

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

如果按 `/ppk/` 子路径把所有资源反代到脚本 gateway `8989`：

```nginx
server {
    listen 80;
    server_name app.example.com;

    location = /ppk {
        return 301 /ppk/;
    }

    location /ppk/ {
        proxy_pass http://127.0.0.1:8989;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Prefix /ppk;
    }
}
```

## 7.4 前后端分离部署

如果前端由服务商托管，backend 使用独立 API 域名：

- frontend 构建环境只设置 `VITE_API_BASE_URL=https://api.example.com/api`
- backend 设置 `ALLOWED_ORIGINS=https://app.example.com`
- DNS / 网关只把 API 流量转到 Go backend
- MySQL 与 agent-service 仍只能由 backend 在本机或私有网络访问

## 7.5 上线前检查

部署前至少确认：

- `go test ./...` 通过
- `agent-service` 约束测试通过，且 `/health` 只在本机或内网可访问
- `npx vue-tsc -b --noEmit` 和 `npm run build` 通过
- `APP_ENV=production` 下 `JWT_SECRET`、`MYSQL_DSN`、`ALLOWED_ORIGINS`、`AGENT_INTERNAL_TOKEN` 都已替换
- 脚本部署时云服务器安全组 / 防火墙只对公网开放 `8989`，MySQL、backend `18989` 和 agent-service 不开放公网
- 前端构建产物中没有 `MYSQL_DSN`、`AGENT_SERVICE_URL`、`AGENT_INTERNAL_TOKEN`、`LLM_API_KEY`
- 如果历史 `.env` 中出现过真实 `LLM_API_KEY`，上线前必须在供应商侧轮换该 key

更完整的部署说明见 `README-DEPLOY.md`。

---

## 8. 本地开发启动

## 8.1 初始化数据库

先创建并导入数据库：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p111111 -e "CREATE DATABASE IF NOT EXISTS ppk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -h 127.0.0.1 -P 3306 -u root -p111111 -e "CREATE USER IF NOT EXISTS 'ppk_dev'@'127.0.0.1' IDENTIFIED BY 'ppk_dev_password'; GRANT SELECT, INSERT, UPDATE, DELETE ON ppk.* TO 'ppk_dev'@'127.0.0.1'; FLUSH PRIVILEGES;"
mysql -h 127.0.0.1 -P 3306 -u root -p111111 ppk < database/schema.sql
mysql -h 127.0.0.1 -P 3306 -u root -p111111 ppk < database/seed.sql
```

## 8.2 启动 backend

建议在 WSL 内执行：

```bash
cd backend
APP_ENV=development \
APP_PORT=8080 \
MYSQL_DSN="ppk_dev:ppk_dev_password@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET="dev-jwt-secret-change-me-32-bytes" \
ALLOWED_ORIGINS="http://127.0.0.1:5173,http://localhost:5173" \
AGENT_SERVICE_URL="http://127.0.0.1:8090" \
AGENT_INTERNAL_TOKEN="dev-agent-internal-token-change-me" \
go run ./cmd/server
```

启动成功后默认提供：

- backend API：`http://127.0.0.1:8080`

## 8.3 启动 frontend

建议在 WSL 内执行：

```bash
cd frontend
npm install
cp .env.example .env.local
npm run dev -- --host 0.0.0.0 --port 5173
```

启动成功后：

- frontend：`http://127.0.0.1:5173`
- API：默认同源 `/api`，由 Vite proxy 转发到 `VITE_DEV_API_PROXY_TARGET`

## 8.4 演示账号

### 管理员
- 账号：`admin`
- 密码：`123456`

### 商家
- 账号：`merchant`
- 密码：`123456`

### 消费者落地页示例
- `http://127.0.0.1:5173/landing/landing-demo-001`

---

## 9. 主要接口说明

## 9.1 商家端
- `POST /api/merchant/auth/login`
- `GET /api/merchant/store/detail`
- `PUT /api/merchant/store/detail`
- `GET /api/merchant/store/keywords`
- `POST /api/merchant/store/keywords`
- `PUT /api/merchant/store/keywords/:id`
- `DELETE /api/merchant/store/keywords/:id`
- `GET /api/merchant/store/images`
- `POST /api/merchant/store/images/upload`
- `DELETE /api/merchant/store/images/:id`
- `GET /api/merchant/store/platform-links`
- `POST /api/merchant/store/platform-links`
- `PUT /api/merchant/store/platform-links/:id`
- `PUT /api/merchant/store/platform-links/:id/status`
- `DELETE /api/merchant/store/platform-links/:id`
- `GET /api/merchant/reviews`
- `POST /api/merchant/reviews`
- `PUT /api/merchant/reviews/:id`
- `DELETE /api/merchant/reviews/:id`
- `POST /api/merchant/reviews/generate`
- `GET /api/merchant/review-generation-tasks`

## 9.2 管理员端
- `POST /api/admin/auth/login`
- `GET /api/admin/merchants`
- `PUT /api/admin/merchants/:id/status`
- `GET /api/admin/stores`
- `PUT /api/admin/stores/:id/status`
- `GET /api/admin/nfc-tags`
- `POST /api/admin/nfc-tags`
- `PUT /api/admin/nfc-tags/:id/bind`
- `PUT /api/admin/nfc-tags/:id/status`
- `GET /api/admin/review-generation-tasks`
- `GET /api/admin/stats`

## 9.3 消费者端
- `GET /api/public/landing/:token/init`：返回门店、图片、关键词、可用平台入口，不发放评价
- `POST /api/public/landing/:token/switch-review`：请求体必须包含 `platformCode`，按所选平台发放或更换评价
- `POST /api/public/landing/:token/events`

---

## 10. 文档说明

当前仓库内可用文档：

- `README.md`：项目结构、当前能力、本地启动与部署主线
- `README-DEPLOY.md`：更完整的生产部署、systemd、Nginx、上线检查与排查
- `agent-service/README.md`：内部 AI 文案生成服务的运行、调用与接入说明

历史 README 中曾引用 `docs/` 产品与开发文档目录；当前仓库未包含该目录，部署时以以上三份文档为准。

---

## 11. 当前验证结论

当前已验证通过：

- `backend` 的 `go test ./...` 通过
- `agent-service` 约束测试通过
- `agent-service` 可编译检查，`/health` 返回最小健康信息
- `frontend` 的 `npx vue-tsc -b --noEmit` 通过
- `frontend` 已改为只通过 `VITE_API_BASE_URL` 访问 Go backend `/api`
- 管理员后台已接入商家、门店、NFC 标签状态操作；商家后台已接入关键词、图片、平台入口与评价的删除/启停操作
- 消费者落地页已改为先选择平台，再按 `platformCode` 从平台隔离的评价池领取文案
- 一键部署脚本会在启动后通过 `8989` 入口运行后台接口 smoke test
- 当前工作区已移除被误提交的 `frontend/node_modules`、`frontend/dist` 与编译产物

部署前仍需在目标服务器或干净构建环境执行完整启动联调：MySQL 初始化、agent-service 启动、backend 启动、frontend `npm ci && npm run build`、商家/管理员/消费者主流程验证。

---

## 12. 后续建议

建议下一步继续完善：

1. 为 `seed.sql` 增加幂等控制，避免重复导入时图片/关键词/评价重复堆积
2. 为 agent-service 增加生产日志、指标、告警、限流与失败重试策略
3. 增加对象存储上传能力
4. 增加自动化浏览器验证（如 Playwright）
5. 增加更完整的管理员审核与统计能力
