# Nexus API Gateway

一个高性能、高可用的 API 聚合与转发系统，作为中间商转售上游 API（官方和逆向工程）给下游用户。

## 🌟 特性

- **高可用性**: 即使单个上游提供商宕机，网关也不会失败
- **精确计费**: 准确的账单、配额管理和速率限制
- **多模态规则**: 按 operation + unit 配置图片/音频/视频的计费与限流规则，并接入调度/扣费链路
- **资源抽象**: 用户调用标准端点，系统根据成本和健康状况决定使用哪个上游提供商
- **OpenAI 兼容**: 完全兼容 OpenAI API 格式
- **流式响应**: 支持 SSE 流式传输
- **多渠道负载均衡**: 支持优先级、轮询、最低延迟等策略

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户请求                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Gateway Layer (网关层)                       │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   认证中间件  │ │  限流中间件  │ │  日志中间件  │ │ 协议转换  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Dispatcher Layer (调度层)                      │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
│  │  路由策略    │ │  故障转移    │ │  模型映射    │               │
│  └─────────────┘ └─────────────┘ └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Resource Pool (资源池)                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
│  │  会话管理    │ │  保活工作    │ │  健康监控    │               │
│  └─────────────┘ └─────────────┘ └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     上游 API 提供商                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
│  │   OpenAI    │ │   Azure     │ │  逆向代理    │               │
│  └─────────────┘ └─────────────┘ └─────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

## 🛠️ 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL
- **缓存**: Redis

### 前端
- **框架**: Next.js 14+ (App Router)
- **UI**: Tailwind CSS + ShadcnUI
- **状态管理**: Zustand
- **HTTP 客户端**: Axios

### 部署
- Docker + Docker Compose
- Nginx (反向代理)
- Prometheus + Grafana (可选监控栈)

## 📁 项目结构

```
/
├── backend/                 # Go 后端
│   ├── cmd/
│   │   ├── api-server/     # 主程序入口
│   │   └── job-worker/     # 异步作业 Worker（generation_tasks）
│   ├── internal/
│   │   ├── config/         # 配置管理
│   │   ├── model/          # 数据模型
│   │   ├── database/       # 数据库连接
│   │   ├── cache/          # Redis 缓存
│   │   ├── middleware/     # 中间件
│   │   ├── repository/     # 数据访问层
│   │   ├── service/        # 业务逻辑层
│   │   └── handler/        # HTTP 处理器
│   ├── Dockerfile
│   ├── Makefile
│   └── go.mod
├── frontend-user/          # 用户端前端
├── frontend-admin/         # 管理员端前端
├── plans/                  # 架构设计文档
├── prometheus/             # Prometheus 配置
├── grafana/                # Grafana Provisioning（数据源/仪表盘）
├── resource-pool/          # Python 资源池（逆向/不稳定上游引擎）
├── docker-compose.yml
└── README.md
```

## 🚀 快速开始

### 默认管理员账号

系统首次启动时会自动创建默认管理员账号：

- **用户名**: `admin`
- **密码**: `admin123456`
- **邮箱**: `admin@example.com`

⚠️ **重要提示**: 首次登录后请立即修改默认密码！

您可以通过环境变量自定义默认管理员账号：

```env
# 推荐（同时支持开发/线上）
ADMIN_USERNAME=your_admin_username
ADMIN_PASSWORD=your_secure_password
ADMIN_EMAIL=your_admin_email

# 兼容旧变量名（仍支持）
# DEFAULT_ADMIN_USERNAME=your_admin_username
# DEFAULT_ADMIN_PASSWORD=your_secure_password
# DEFAULT_ADMIN_EMAIL=your_admin_email
```

### 监控（可选）

`docker-compose.yml` 默认包含 Prometheus + Grafana：

- Prometheus: `http://localhost:${PROMETHEUS_PORT:-9090}`
- Grafana: `http://localhost:${GRAFANA_PORT:-3002}`（默认账号 `admin` / `admin`）
- Backend `/metrics`: `http://localhost:${API_PORT:-8080}/metrics`
- Grafana 数据源：已通过 `grafana/provisioning/datasources/prometheus.yml` 自动接入 Prometheus

### 前置要求

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### 使用 Docker Compose 启动

```bash
# 克隆项目
git clone <repository-url>
cd nexus-api-gateway

# 复制环境变量文件
cp .env.example .env

# 编辑环境变量
vim .env

# 启动后端服务（postgres/redis/api/resource-pool）
docker compose up -d

# （可选）启动完整栈（含前端容器 + nginx）
docker compose --profile production up -d

# 查看日志
docker compose logs -f
```

### 本地开发

#### 后端

```bash
cd backend

# 安装依赖
go mod download

# 复制环境变量
cp .env.example .env

# 终端 1：运行 API 服务器
make dev

# 终端 2：运行异步作业 Worker（处理 generation_tasks）
make dev-worker

# 或者构建后分别运行
make run
make run-worker
```

#### 前端用户端

```bash
cd frontend-user

# 安装依赖
npm install

# 运行开发服务器
npm run dev
```

#### 前端管理员端

```bash
cd frontend-admin

# 安装依赖
npm install

# 运行开发服务器
npm run dev
```

## 📖 API 文档

### Gateway API (OpenAI 兼容)

| 端点 | 方法 | 描述 |
|------|------|------|
| `/v1/chat/completions` | POST | 聊天补全 |
| `/v1/completions` | POST | 文本补全 |
| `/v1/embeddings` | POST | 文本嵌入 |
| `/v1/models` | GET | 获取可用模型列表 |

### 用户 API

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/user/register` | POST | 用户注册 |
| `/api/user/login` | POST | 用户登录 |
| `/api/user/profile` | GET | 获取用户信息 |
| `/api/user/tokens` | GET/POST | API Key 管理 |
| `/api/user/logs` | GET | 使用日志 |
| `/api/user/orders` | GET/POST | 订单管理 |

### 管理员 API

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/admin/users` | GET/PUT | 用户管理 |
| `/api/admin/channels` | CRUD | 渠道管理 |
| `/api/admin/models` | CRUD | 模型管理 |
| `/api/admin/logs` | GET | 日志查询 |
| `/api/admin/stats` | GET | 统计数据 |

## 🔧 配置说明

### 环境变量

```env
# 服务器配置
SERVER_PORT=8080
SERVER_MODE=debug

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=nexus_api

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT 配置
JWT_SECRET=your-super-secret-key
JWT_EXPIRE_HOURS=24

# 速率限制
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60

# 存储驱动：local 或 oss
STORAGE_DRIVER=local
STORAGE_LOCAL_DIR=./data/objects
# 可选：本地签名 URL 密钥（默认复用 JWT_SECRET）
STORAGE_SIGN_SECRET=

# OSS 对象存储（STORAGE_DRIVER=oss 时生效）
OSS_ENDPOINT=oss-cn-shenzhen.aliyuncs.com
OSS_BUCKET=wool-good
OSS_ACCESS_KEY_ID=<set-in-runtime>
OSS_ACCESS_KEY_SECRET=<set-in-runtime>
# 可选：自定义/CNAME 域名（用于让签名 URL 走你的域名）
OSS_PUBLIC_BASE_URL=https://wool-good.cn-shenzhen.taihangwkz.cn
# 签名 URL 过期时间（秒）
OSS_SIGN_EXPIRE_SECONDS=900
```

### 资源（图片/视频/文件）前缀与访问方式

对象 Key 规则在 `backend/internal/service/asset_key.go`：

- 网站素材（管理员上传）：`site/{kind}/yyyy/mm/dd/{uuid}{ext}`
- 用户测试上传：`users/{user_id}/uploads/{kind}/yyyy/mm/dd/{uuid}{ext}`
- API 调用产物：`users/{user_id}/outputs/{kind}/yyyy/mm/dd/{uuid}{ext}`

私有桶访问通过签名 URL（不要把签名 URL 当作永久地址存库）：

- 网站素材公开访问：`GET /assets/{id}`（302 跳转到签名 URL）
- 用户私有资源：`GET /api/user/assets/{id}/url`（返回签名 URL）或 `GET /api/user/assets/{id}`（302 跳转）

## 📊 数据库模型

### 核心表

- **users**: 用户信息
- **tokens**: API Key
- **channels**: 上游渠道
- **models**: AI 模型定义
- **logs**: 请求日志
- **orders**: 充值订单
- **resource_accounts**: 逆向工程账户
- **announcements**: 系统公告

## 🔐 安全特性

- JWT 认证
- API Key 认证 (Bearer Token)
- 速率限制 (滑动窗口 + 令牌桶)
- IP 白名单
- 请求签名验证

## 📈 监控与日志

- 结构化日志 (JSON 格式)
- 请求追踪 (Request ID)
- 性能指标收集
- 健康检查端点

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

MIT License

## 📞 联系方式

如有问题或建议，请提交 Issue 或联系维护者。
