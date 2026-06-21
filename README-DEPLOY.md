# README-DEPLOY

## 1. 文档目的

本文档用于说明 **PPK NFC 评价系统 MVP** 的部署方式，覆盖以下内容：

- MySQL 初始化
- backend 部署与启动
- frontend 部署与启动
- 环境变量说明
- 常见部署方案
- 上线前检查项
- 常见问题排查

本文档面向开发、测试、运维与部署人员。

---

## 2. 项目组成

系统由四部分组成：

1. **MySQL 数据库**
   - 使用 `database/schema.sql` 初始化表结构
   - 使用 `database/seed.sql` 初始化演示数据

2. **backend 后端服务**
   - 技术栈：Gin + Gorm + JWT
   - 默认监听端口：`8080`
   - 对浏览器开放的唯一业务 API 入口为 `/api/*`

3. **frontend 前端服务**
   - 技术栈：Vue 3 + Vite + Pinia + Axios
   - 开发默认端口：`5173`
   - 生产环境建议构建静态资源后由 Nginx 托管

4. **agent-service 内部服务**
   - 供 backend 在服务器本机或私有网络内调用
   - 不对浏览器开放，不写入前端环境变量

云端推荐拓扑：

```text
Browser
  -> public gateway :8989
  -> local Go backend 127.0.0.1:18989 (/api)
  -> local/private MySQL
  -> local/private agent-service
```

浏览器永远只访问 `8989` 提供的前端静态资源和同源 `/api`。Go backend 只绑定本机 `127.0.0.1:18989`，MySQL 与 agent-service 不开放公网，不配置到任何前端 `.env` 文件中。

---

## 3. 部署前准备

## 3.1 环境要求

建议环境：

- Linux / WSL / macOS
- Go `1.18+`
- Node.js `18+`
- npm `9+`
- MySQL `8.0+`

> 当前项目已在 WSL + MySQL 8.0 + Go 1.18 + Node 22 环境下完成基础验证。

## 3.2 目录说明

部署时涉及的关键目录：

```text
backend/      # 后端服务
frontend/     # 前端项目
database/     # 数据库初始化脚本
scripts/      # 一键部署脚本与 public gateway
```

---

## 3.3 一键部署入口

默认推荐使用仓库内脚本完成依赖安装、构建与启动：

```bash
cp .env.deploy.example .env.deploy
# 编辑 .env.deploy，至少确认 MYSQL_DSN 可连接、LLM_API_KEY 已填写
scripts/deploy.sh start
```

脚本启动后的端口约束：

| 组件 | 监听地址 | 是否对公网开放 |
|---|---|---|
| public gateway / frontend | `0.0.0.0:8989` | 是 |
| Go backend | `127.0.0.1:18989` | 否 |
| agent-service | `127.0.0.1:8090` | 否 |
| MySQL | `127.0.0.1:3306` 或内网地址 | 否 |

常用命令：

```bash
scripts/deploy.sh status
scripts/deploy.sh logs
scripts/deploy.sh restart
scripts/deploy.sh stop
```

`scripts/deploy.sh start` 会执行：

- 启动前预检 `LLM_API_KEY`
- 从 `MYSQL_DSN` 解析 MySQL host/port 并预检 TCP 连通性
- `go mod download`
- `npm ci`
- `python3 -m venv agent-service/.venv`
- `pip install -r agent-service/requirements.txt`
- `VITE_API_BASE_URL=/api npm run build`
- `go build -o .deploy/bin/ppk-server ./cmd/server`
- 后台启动 agent-service、backend、public gateway

运行日志和 PID 文件位于 `.deploy/`。`JWT_SECRET` 与 `AGENT_INTERNAL_TOKEN` 未配置时会自动生成到 `.deploy/runtime.env`。

如果 `LLM_API_KEY` 为空，脚本默认会停止，因为 agent-service 虽可探活但生成会返回 503。仅在需要先启动 UI/API、不启用 AI 生成时，才设置：

```bash
ALLOW_EMPTY_LLM_KEY=true
```

如果希望脚本初始化 MySQL，可在 `.env.deploy` 中设置：

```bash
INIT_DB=true
LOAD_SEED=false
MYSQL_ROOT_USER=root
MYSQL_ROOT_PASSWORD=<root-password>
DB_APP_PASSWORD=<strong-db-password>
```

生产环境防火墙只需要开放 `8989`。不要开放 `18989`、`8090` 或 MySQL 端口。

---

## 4. MySQL 部署与初始化

## 4.1 创建数据库

使用以下命令创建数据库：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p111111 -e "CREATE DATABASE IF NOT EXISTS ppk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -h 127.0.0.1 -P 3306 -u root -p111111 -e "CREATE USER IF NOT EXISTS 'ppk_dev'@'127.0.0.1' IDENTIFIED BY 'ppk_dev_password'; GRANT SELECT, INSERT, UPDATE, DELETE ON ppk.* TO 'ppk_dev'@'127.0.0.1'; FLUSH PRIVILEGES;"
```

如果你的 MySQL 用户名、密码或地址不同，请自行替换。

## 4.2 导入表结构

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p111111 ppk < database/schema.sql
```

## 4.3 导入初始化数据

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p111111 ppk < database/seed.sql
```

## 4.4 导入后检查

建议执行：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p111111 -D ppk -e "SHOW TABLES;"
```

预期应至少看到：

- `admin_users`
- `merchant_users`
- `stores`
- `store_keywords`
- `store_images`
- `store_platform_links`
- `review_items`
- `review_display_logs`
- `review_generation_tasks`
- `nfc_tags`

---

## 5. backend 部署

## 5.1 配置说明

backend 通过环境变量读取配置：

| 变量名 | 说明 | 示例 |
|---|---|---|
| `APP_HOST` | 后端监听地址，脚本部署固定为本机 | `127.0.0.1` |
| `APP_PORT` | 后端监听端口 | `8080` |
| `APP_ENV` | 运行环境，生产使用 `production` | `production` |
| `MYSQL_DSN` | MySQL 连接串，使用最小权限账号 | `ppk_app:<password>@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local` |
| `JWT_SECRET` | JWT 密钥，至少 32 字符 | `<random-32-plus-char-secret>` |
| `ALLOWED_ORIGINS` | 允许访问 API 的前端 origin，逗号分隔 | `https://app.example.com` |
| `AGENT_SERVICE_URL` | backend 内部调用 agent-service 的地址 | `http://127.0.0.1:8090` |
| `AGENT_INTERNAL_TOKEN` | backend 与 agent-service 共享的内部令牌 | `<random-32-plus-char-token>` |

> 当前实现中，backend 启动时**不自动建表**，默认依赖 `database/schema.sql` 已经执行完成。
> 生产环境会校验 `APP_ENV=production` 下的关键配置，缺少强 `JWT_SECRET`、`MYSQL_DSN`、`ALLOWED_ORIGINS` 或 `AGENT_INTERNAL_TOKEN` 会拒绝启动。

## 5.2 本地直接启动

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

## 5.3 构建二进制后启动

```bash
cd backend
go build -o ppk-server ./cmd/server

APP_ENV=production \
APP_HOST=127.0.0.1 \
APP_PORT=18989 \
MYSQL_DSN="ppk_app:<password>@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET="<random-32-plus-char-secret>" \
ALLOWED_ORIGINS="https://app.example.com" \
AGENT_SERVICE_URL="http://127.0.0.1:8090" \
AGENT_INTERNAL_TOKEN="<random-32-plus-char-token>" \
./ppk-server
```

## 5.4 systemd 部署示例（Linux）

可创建 `/etc/systemd/system/ppk-backend.service`：

```ini
[Unit]
Description=PPK Backend Service
After=network.target mysql.service

[Service]
WorkingDirectory=/opt/ppk/backend
Environment="APP_ENV=production"
Environment="APP_HOST=127.0.0.1"
Environment="APP_PORT=18989"
Environment="MYSQL_DSN=ppk_app:<password>@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"
Environment="JWT_SECRET=<random-32-plus-char-secret>"
Environment="ALLOWED_ORIGINS=https://app.example.com"
Environment="AGENT_SERVICE_URL=http://127.0.0.1:8090"
Environment="AGENT_INTERNAL_TOKEN=<random-32-plus-char-token>"
ExecStart=/opt/ppk/backend/ppk-server
Restart=always
RestartSec=3
User=www-data

[Install]
WantedBy=multi-user.target
```

然后执行：

```bash
sudo systemctl daemon-reload
sudo systemctl enable ppk-backend
sudo systemctl start ppk-backend
sudo systemctl status ppk-backend
```

---

## 5.5 agent-service 部署

agent-service 是内部 AI 文案生成服务，只供 backend 在服务器本机或私有网络内调用。不要把它配置到前端环境变量，也不要在 Nginx 或 API 网关中暴露 `/generate-reviews`。

关键环境变量：

| 变量名 | 说明 | 示例 |
|---|---|---|
| `LLM_API_KEY` | LLM 供应商 API key，只保存在服务器端 | `<your-llm-api-key>` |
| `LLM_BASE_URL` | OpenAI 兼容端点，可按供应商调整 | `https://api.openai.com/v1` |
| `LLM_MODEL` | 文案生成模型 | `gpt-5.4` |
| `AGENT_HOST` | 监听地址，生产默认保持本机 | `127.0.0.1` |
| `AGENT_PORT` | 监听端口 | `8090` |
| `AGENT_INTERNAL_TOKEN` | 与 backend 共享的内部令牌 | `<same-token-as-backend>` |

启动示例：

```bash
cd agent-service
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
cp .env.example .env

# 编辑 .env 后启动
python -m app.main
```

本机健康检查：

```bash
curl http://127.0.0.1:8090/health
```

如果使用 systemd，可创建 `/etc/systemd/system/ppk-agent.service`：

```ini
[Unit]
Description=PPK Agent Service
After=network.target

[Service]
WorkingDirectory=/opt/ppk/agent-service
EnvironmentFile=/opt/ppk/agent-service/.env
ExecStart=/opt/ppk/agent-service/.venv/bin/python -m app.main
Restart=always
RestartSec=3
User=www-data

[Install]
WantedBy=multi-user.target
```

然后执行：

```bash
sudo systemctl daemon-reload
sudo systemctl enable ppk-agent
sudo systemctl start ppk-agent
sudo systemctl status ppk-agent
```

---

## 6. frontend 部署

## 6.1 安装依赖

```bash
cd frontend
npm install
```

## 6.2 开发环境启动

```bash
cd frontend
cp .env.example .env.local
npm run dev -- --host 0.0.0.0 --port 5173
```

开发环境访问地址：

- `http://127.0.0.1:5173`

`frontend/.env.example` 只包含允许暴露给浏览器的变量：

```bash
VITE_API_BASE_URL=http://127.0.0.1:8080/api
```

生产环境不配置时，前端默认请求同源 `/api`。脚本部署时会用 `VITE_API_BASE_URL=/api` 构建前端，并由 `8989` public gateway 转发到本机 backend `127.0.0.1:18989`。不要把 MySQL、agent-service 或任何服务端密钥写入前端环境变量。

## 6.3 生产构建

```bash
cd frontend
VITE_API_BASE_URL=/api npm run build
```

构建完成后，产物输出到：

- `frontend/dist/`

## 6.4 使用 Nginx 托管前端静态资源

示例配置：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    root /opt/ppk/frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

生产默认推荐同源 `/api` 反向代理。脚本部署已内置 public gateway；如果改用 Nginx，同样只把公网流量转给本机 backend，MySQL 与 agent-service 仍保持本机或私有网络访问。

---

## 7. 推荐部署拓扑

## 7.1 简单部署（单机）

适合 MVP 演示环境：

- MySQL：本机
- backend：本机 18989
- frontend / gateway：本机 8989

```text
Browser
   └── public gateway :8989
        ├── frontend(dist)
        └── /api -> backend 127.0.0.1:18989
                   ├── private MySQL:3306
                   └── private agent-service:8090
```

单机脚本部署只对公网暴露 `8989`。MySQL、agent-service、backend 的内部端口绑定 `127.0.0.1` 或内网地址，不直接公开。

## 7.2 推荐部署（前后端分离）

- frontend：静态托管服务或 Nginx 托管静态文件
- backend：systemd 或容器运行，对外提供 `/api`
- MySQL：本机、内网或托管数据库，仅 backend 可访问
- agent-service：本机或内网服务，仅 backend 可访问

浏览器侧部署产物中不包含数据库地址、agent-service 地址或服务端密钥。

---

## 8. 反向代理建议

如果使用 Nginx 做前后端统一入口，可考虑：

```nginx
server {
    listen 80;
    server_name your-domain.com;

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

这样 frontend 访问 `/api/*` 时会自动转发到 backend。脚本部署已经用 `scripts/serve_gateway.py` 实现了等价能力，固定公开端口为 `8989`。

---

## 9. 演示账号与测试入口

## 9.1 管理员账号

- 账号：`admin`
- 密码：`123456`

## 9.2 商家账号

- 账号：`merchant`
- 密码：`123456`

## 9.3 消费者演示页面

- `http://127.0.0.1:5173/landing/landing-demo-001`

如果走 Nginx 托管，则替换为你的正式域名。

---

## 10. 上线前检查清单

部署前建议至少确认：

### 10.1 数据库
- [ ] `ppk` 数据库已创建
- [ ] `schema.sql` 已执行
- [ ] `seed.sql` 已执行（仅演示环境）
- [ ] MySQL 账号有读写权限

### 10.2 backend
- [ ] `go build ./...` 能通过
- [ ] `MYSQL_DSN` 正确
- [ ] `JWT_SECRET` 已替换为生产值
- [ ] `ALLOWED_ORIGINS` 只包含真实前端域名
- [ ] `APP_HOST=127.0.0.1`
- [ ] `APP_PORT=18989`
- [ ] `AGENT_SERVICE_URL` 指向本机或私有网络 agent-service
- [ ] `AGENT_INTERNAL_TOKEN` 与 agent-service 一致
- [ ] 服务可启动并监听端口

### 10.3 agent-service
- [ ] `AGENT_HOST=127.0.0.1` 或私有网络地址
- [ ] `AGENT_INTERNAL_TOKEN` 已替换为生产值
- [ ] `LLM_API_KEY` 只存在于服务器端
- [ ] `/health` 仅在本机或内网可访问
- [ ] `/generate-reviews` 未通过 Nginx 或 API 网关暴露给浏览器

### 10.4 frontend
- [ ] `npm install` 成功
- [ ] `npm run build` 成功
- [ ] `dist/` 已发布到静态服务器
- [ ] 路由刷新不会 404（需 `try_files /index.html`）
- [ ] 构建环境只包含 `VITE_API_BASE_URL` 这类浏览器可见变量
- [ ] 脚本部署时只开放 `8989`，不开放 `18989`、`8090`、`3306`

### 10.5 联通性
- [ ] 商家登录可用
- [ ] 管理员登录可用
- [ ] 消费者落地页初始化可用
- [ ] `switch-review` 可用
- [ ] `events` 可用

---

## 11. 常见问题

## 11.1 backend 启动报数据库权限错误

示例：

```text
Access denied for user 'root'@'localhost'
```

处理方式：
- 检查 backend 服务器上的 `MYSQL_DSN`
- 检查用户名/密码
- 检查数据库是否已创建
- 检查 MySQL 是否允许当前主机连接

## 11.2 backend 启动报表结构冲突

如果同时使用 `schema.sql` 和 `AutoMigrate`，可能会出现表结构冲突。

当前项目已采用：
- **只使用 SQL 脚本建表**
- backend 启动时**不自动迁移**

如果再次改回自动迁移，需要重新评估与现有 SQL 脚本的一致性。

## 11.3 frontend 启动时报 esbuild / UNC 路径问题

在 Windows + WSL 混合环境下，直接从 UNC 路径启动前端可能失败。

建议：
- 在 WSL 内进入 Linux 路径执行 `npm install` 与 `npm run dev`

## 11.4 页面打开但接口失败

检查项：
- backend 是否已启动
- frontend 的 `VITE_API_BASE_URL` 是否指向 Go backend 公开 `/api`，未配置时是否有同源 `/api` 反向代理
- 是否存在跨域问题
- Nginx 是否正确代理 `/api/`
- 不要尝试从浏览器直接请求 MySQL 或 agent-service

## 11.5 backend 调用 agent-service 返回 401

检查项：
- backend 的 `AGENT_INTERNAL_TOKEN` 是否与 agent-service 的 `AGENT_INTERNAL_TOKEN` 完全一致
- backend 请求是否发送了 `X-Agent-Internal-Token`
- agent-service 是否只在本机或私有网络监听
- 不要为了排查问题把 agent-service 暴露到公网

---

## 12. 生产环境建议

当前项目为 MVP，生产部署前建议继续完善：

1. 为 agent-service 增加生产日志、指标、告警、限流与失败重试策略
2. 为 LLM 调用增加成本监控、超时预算、供应商故障降级与 key 轮换流程
3. 把图片能力改为对象存储上传
4. 增加 backend 运行环境文件或 secret manager 接入，避免在 systemd 文件中长期明文维护 secret
5. 增加 HTTPS
6. 增加数据库备份策略
7. 增加前端、后端与 agent-service 的容器化部署

---

## 13. 建议后续补充文件

后续可继续补充：

- `backend/.env.example`
- `docker-compose.yml`
- `Dockerfile.backend`
- `Dockerfile.frontend`
- `Dockerfile.agent-service`
- `README-OPS.md`
- `README-API.md`

---

## 14. 当前已验证通过的部署级事实

当前已经真实验证通过：

- `backend` 的 `go test ./...` 通过
- `agent-service` 约束测试通过
- `agent-service` 可编译检查，`/health` 返回最小健康信息
- `frontend` 的 `npx vue-tsc -b --noEmit` 通过
- frontend 只配置 `VITE_API_BASE_URL`，不会把 MySQL、agent-service 或服务端 secret 暴露到浏览器
- Go backend 主生成器通过 `AGENT_SERVICE_URL` 调用内部 agent-service，mock 只保留为空池兜底
- frontend 被误提交的 `node_modules`、`dist` 与编译产物已从 git 跟踪中移除

部署到云服务器前，还需要在目标环境完成一次端到端联调：MySQL 导入、agent-service 携真实 `LLM_API_KEY` 启动、backend 携生产环境变量启动、frontend 在干净依赖环境执行 `npm ci && npm run build`，再验证商家、管理员与消费者主流程。
