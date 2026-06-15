# PPK NFC 评价系统 MVP

基于 **Gin + Vue 3 + MySQL** 的 NFC 碰碰卡评价辅助系统 MVP。

本项目面向线下商家：消费者到店后通过手机碰一碰 NFC 标签，打开落地页，获取推荐评价文案、图片素材，并跳转到目标平台进行评论或发布内容。

当前仓库包含：

- 后端服务：Gin + Gorm + JWT
- 前端应用：Vue 3 + Vite + Pinia + Axios
- 数据库脚本：MySQL schema / seed
- 产品与开发文档：位于 `docs/`

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
│   └── seed.sql
├── docs/                       # 产品与开发文档
│   ├── nfc-review-card-prd.md
│   └── development/
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
- `MYSQL_DSN`
- `JWT_SECRET`

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

- `review_generator_mock.go`
  - 当前使用 mock 文案生成器
  - 后续可替换成真实 AI 服务实现

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
  - 文案展示、复制、换一换、平台入口跳转

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
- mock AI 自动生成评价
- JWT 鉴权

### 尚未完善
- 真正的 AI 服务接入
- 对象存储上传
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

### 4.2 平台入口
- 平台入口由商家配置
- 消费者点击入口后可跳转到目标平台
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

- Go：`1.18+`
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
| `APP_PORT` | 后端监听端口 | `8080` |
| `MYSQL_DSN` | MySQL 连接串 | `root:root@tcp(127.0.0.1:3306)/ppk?...` |
| `JWT_SECRET` | JWT 密钥 | `ppk-dev-secret` |

建议启动时显式传入：

```bash
APP_PORT=8080
MYSQL_DSN="root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"
JWT_SECRET="ppk-dev-secret"
```

---

## 7. 使用说明

## 7.1 初始化数据库

先创建并导入数据库：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p111111 -e "CREATE DATABASE IF NOT EXISTS ppk CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -h 127.0.0.1 -P 3306 -u root -p111111 ppk < database/schema.sql
mysql -h 127.0.0.1 -P 3306 -u root -p111111 ppk < database/seed.sql
```

## 7.2 启动 backend

建议在 WSL 内执行：

```bash
cd backend
APP_PORT=8080 \
MYSQL_DSN="root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET="ppk-dev-secret" \
go run ./cmd/server
```

启动成功后默认提供：

- backend API：`http://127.0.0.1:8080`

## 7.3 启动 frontend

建议在 WSL 内执行：

```bash
cd frontend
npm install
npm run dev -- --host 0.0.0.0 --port 5173
```

启动成功后：

- frontend：`http://127.0.0.1:5173`

## 7.4 演示账号

### 管理员
- 账号：`admin`
- 密码：`123456`

### 商家
- 账号：`merchant`
- 密码：`123456`

### 消费者落地页示例
- `http://127.0.0.1:5173/landing/landing-demo-001`

---

## 8. 主要接口说明

## 8.1 商家端
- `POST /api/merchant/auth/login`
- `GET /api/merchant/store/detail`
- `PUT /api/merchant/store/detail`
- `GET /api/merchant/store/keywords`
- `POST /api/merchant/store/keywords`
- `GET /api/merchant/store/images`
- `POST /api/merchant/store/images/upload`
- `GET /api/merchant/store/platform-links`
- `POST /api/merchant/store/platform-links`
- `GET /api/merchant/reviews`
- `POST /api/merchant/reviews`
- `POST /api/merchant/reviews/generate`
- `GET /api/merchant/review-generation-tasks`

## 8.2 管理员端
- `POST /api/admin/auth/login`
- `GET /api/admin/merchants`
- `GET /api/admin/stores`
- `GET /api/admin/nfc-tags`
- `POST /api/admin/nfc-tags`
- `PUT /api/admin/nfc-tags/:id/bind`
- `GET /api/admin/review-generation-tasks`
- `GET /api/admin/stats`

## 8.3 消费者端
- `GET /api/public/landing/:token/init`
- `POST /api/public/landing/:token/switch-review`
- `POST /api/public/landing/:token/events`

---

## 9. 文档说明

### 产品文档
- `docs/nfc-review-card-prd.md`

### 开发文档拆分
- `docs/development/README.md`
- `docs/development/01-consumer-h5.md`
- `docs/development/02-merchant-console.md`
- `docs/development/03-admin-console.md`
- `docs/development/04-review-pool-and-ai.md`
- `docs/development/05-data-model-and-api.md`
- `docs/development/06-nonfunctional-and-ops.md`

---

## 10. 当前验证结论

当前已验证通过：

- backend 可构建并可启动
- frontend 可安装依赖并可构建 / 可启动 dev server
- MySQL 可初始化并导入演示数据
- 商家登录 API 可用
- 管理员登录 API 可用
- 消费者初始化 API 可用
- `switch-review` API 可用
- `events` API 可用
- frontend 与 backend 已联通，页面能返回真实 HTML / JS 资源

---

## 11. 后续建议

建议下一步继续完善：

1. 为 `seed.sql` 增加幂等控制，避免重复导入时图片/关键词/评价重复堆积
2. 为 frontend 增加更完整的页面交互与错误提示
3. 接入真实 AI 服务，替换 mock 评价生成器
4. 增加对象存储上传能力
5. 增加自动化浏览器验证（如 Playwright）
6. 增加更完整的管理员审核与统计能力
