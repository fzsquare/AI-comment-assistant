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

系统由三部分组成：

1. **MySQL 数据库**
   - 使用 `database/schema.sql` 初始化表结构
   - 使用 `database/seed.sql` 初始化演示数据

2. **backend 后端服务**
   - 技术栈：Gin + Gorm + JWT
   - 默认监听端口：`8080`

3. **frontend 前端服务**
   - 技术栈：Vue 3 + Vite + Pinia + Axios
   - 开发默认端口：`5173`
   - 生产环境建议构建静态资源后由 Nginx 托管

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
```

---

## 4. MySQL 部署与初始化

## 4.1 创建数据库

使用以下命令创建数据库：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p111111 -e "CREATE DATABASE IF NOT EXISTS ppk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
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
| `APP_PORT` | 后端监听端口 | `8080` |
| `MYSQL_DSN` | MySQL 连接串 | `root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local` |
| `JWT_SECRET` | JWT 密钥 | `ppk-dev-secret` |

> 当前实现中，backend 启动时**不自动建表**，默认依赖 `database/schema.sql` 已经执行完成。

## 5.2 本地直接启动

```bash
cd backend
APP_PORT=8080 \
MYSQL_DSN="root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET="ppk-dev-secret" \
go run ./cmd/server
```

## 5.3 构建二进制后启动

```bash
cd backend
go build -o ppk-server ./cmd/server

APP_PORT=8080 \
MYSQL_DSN="root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET="ppk-dev-secret" \
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
Environment="APP_PORT=8080"
Environment="MYSQL_DSN=root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"
Environment="JWT_SECRET=ppk-prod-secret"
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

## 6. frontend 部署

## 6.1 安装依赖

```bash
cd frontend
npm install
```

## 6.2 开发环境启动

```bash
cd frontend
npm run dev -- --host 0.0.0.0 --port 5173
```

开发环境访问地址：

- `http://127.0.0.1:5173`

## 6.3 生产构建

```bash
cd frontend
npm run build
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

如果前端和后端不在同域，需要处理跨域或配置反向代理。

---

## 7. 推荐部署拓扑

## 7.1 简单部署（单机）

适合 MVP 演示环境：

- MySQL：本机
- backend：本机 8080
- frontend：本机 Nginx 或 Vite

```text
Nginx / 浏览器
   ├── frontend(dist)
   └── reverse proxy -> backend:8080
                     -> MySQL:3306
```

## 7.2 推荐部署（前后端分离）

- frontend：Nginx 托管静态文件
- backend：systemd 或容器运行
- MySQL：独立数据库实例

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
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

这样 frontend 访问 `/api/*` 时会自动转发到 backend。

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
- [ ] 服务可启动并监听端口

### 10.3 frontend
- [ ] `npm install` 成功
- [ ] `npm run build` 成功
- [ ] `dist/` 已发布到静态服务器
- [ ] 路由刷新不会 404（需 `try_files /index.html`）

### 10.4 联通性
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
- 检查 `MYSQL_DSN`
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
- frontend 中 `src/api/http.ts` 的 `baseURL` 是否正确
- 是否存在跨域问题
- Nginx 是否正确代理 `/api/`

---

## 12. 生产环境建议

当前项目为 MVP，生产部署前建议继续完善：

1. 将 mock AI 生成器替换为真实 AI 服务
2. 把图片能力改为对象存储上传
3. 增加 `.env` / `.env.production` 配置管理
4. 增加日志落盘与监控告警
5. 增加 HTTPS
6. 增加数据库备份策略
7. 增加前端与后端的容器化部署

---

## 13. 建议后续补充文件

后续可继续补充：

- `.env.example`
- `docker-compose.yml`
- `Dockerfile.backend`
- `Dockerfile.frontend`
- `README-OPS.md`
- `README-API.md`

---

## 14. 当前已验证通过的部署级事实

当前已经真实验证通过：

- MySQL 数据库可创建并导入脚本
- backend 可连接数据库并启动
- frontend 可安装依赖、构建、启动 dev server
- 商家登录 API 可用
- 管理员登录 API 可用
- 消费者初始化 API 可用
- `switch-review` API 可用
- `events` API 可用
- frontend 与 backend 已完成基础联通

这说明当前 MVP 已具备基础部署与演示条件。
