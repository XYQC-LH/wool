# Nexus API 管理后台

高性能 API 聚合与转发平台的管理后台，基于 Next.js 14、React 18、TypeScript 和 Tailwind CSS 构建。

## 功能特性

### 核心功能
- 📊 **仪表盘** - 系统运行状态概览，包含统计数据和图表
- 👥 **用户管理** - 管理用户账户、余额、状态
- 🔌 **渠道管理** - 管理上游 API 渠道，支持测试和健康检查
- 🤖 **模型管理** - 管理 AI 模型和定价
- 📝 **日志查询** - 查看 API 请求日志和统计
- 💳 **订单管理** - 管理用户充值订单
- 🗄 **资源账户** - 管理逆向工程账户和资源池
- 📢 **公告管理** - 发布和管理系统公告
- ⚙️ **系统设置** - 配置系统参数和安全设置

### 技术特性
- ⚡ **高性能** - 基于 Next.js 14 App Router
- 🎨 **现代化 UI** - 使用 Tailwind CSS 和 Radix UI
- 🌙 **深色主题** - 默认深色主题，支持主题切换
- 📱 **响应式设计** - 完美支持移动端和桌面端
- 🔒 **安全认证** - JWT Token 认证
- 🔄 **实时更新** - Zustand 状态管理
- 🎯 **TypeScript** - 完整的类型安全
- 🧩 **组件化** - 可复用的 UI 组件库

## 技术栈

### 前端框架
- **Next.js 14.1.0** - React 框架
- **React 18.2.0** - UI 库
- **TypeScript 5.3.3** - 类型系统

### UI 组件库
- **Radix UI** - 无障碍的 UI 原语组件
  - @radix-ui/react-dialog
  - @radix-ui/react-dropdown-menu
  - @radix-ui/react-label
  - @radix-ui/react-select
  - @radix-ui/react-slot
  - @radix-ui/react-switch
  - @radix-ui/react-tabs
  - @radix-ui/react-toast
  - @radix-ui/react-tooltip
- **Lucide React** - 图标库
- **Recharts** - 图表库

### 样式和工具
- **Tailwind CSS 3.4.1** - CSS 框架
- **clsx** - 条件类名工具
- **tailwind-merge** - Tailwind 类名合并
- **class-variance-authority** - 组件变体管理

### 状态管理和数据
- **Zustand 4.5.0** - 轻量级状态管理
- **Axios 1.6.5** - HTTP 客户端
- **js-cookie 3.0.5** - Cookie 管理
- **date-fns 3.3.0** - 日期处理

## 项目结构

```
frontend-admin/
├── src/
│   ├── app/                    # Next.js App Router 页面
│   │   ├── dashboard/          # 管理后台页面
│   │   │   ├── page.tsx      # 仪表盘
│   │   │   ├── users/        # 用户管理
│   │   │   ├── channels/      # 渠道管理
│   │   │   ├── models/        # 模型管理
│   │   │   ├── logs/          # 日志查询
│   │   │   ├── orders/        # 订单管理
│   │   │   ├── resources/     # 资源账户
│   │   │   ├── announcements/ # 公告管理
│   │   │   └── settings/      # 系统设置
│   │   ├── login/            # 登录页面
│   │   ├── layout.tsx        # 根布局
│   │   └── globals.css       # 全局样式
│   ├── components/             # React 组件
│   │   ├── ui/              # UI 基础组件
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── input.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── toast.tsx
│   │   │   ├── select.tsx
│   │   │   ├── switch.tsx
│   │   │   └── label.tsx
│   │   ├── common/           # 通用组件
│   │   │   ├── pagination.tsx
│   │   │   ├── empty-state.tsx
│   │   │   ├── loading.tsx
│   │   │   ├── confirm-dialog.tsx
│   │   │   └── error-boundary.tsx
│   │   └── layout/           # 布局组件
│   │       ├── header.tsx
│   │       └── sidebar.tsx
│   ├── lib/                  # 工具函数
│   │   ├── api.ts           # API 客户端
│   │   ├── utils.ts         # 通用工具
│   │   └── validation.ts    # 表单验证
│   └── store/                # 状态管理
│       └── auth.ts          # 认证状态
├── public/                   # 静态资源
├── .env.example             # 环境变量示例
├── next.config.js           # Next.js 配置
├── tailwind.config.ts       # Tailwind 配置
├── tsconfig.json           # TypeScript 配置
└── package.json            # 项目依赖
```

## 快速开始

### 环境要求
- Node.js >= 18.0.0
- npm >= 9.0.0 或 pnpm >= 8.0.0

### 安装依赖

```bash
# 使用 npm
npm install

# 使用 pnpm
pnpm install

# 使用 yarn
yarn install
```

### 环境配置

复制 `.env.example` 到 `.env` 并配置环境变量：

```bash
cp .env.example .env
```

主要环境变量：

```env
# API 地址
NEXT_PUBLIC_API_URL=http://localhost:8080

# 应用名称
NEXT_PUBLIC_APP_NAME=Nexus Admin

# 应用 URL
NEXT_PUBLIC_APP_URL=http://localhost:3001

# 默认时区
NEXT_PUBLIC_DEFAULT_TIMEZONE=Asia/Shanghai

# 默认语言
NEXT_PUBLIC_DEFAULT_LANGUAGE=zh-CN
```

### 开发模式

```bash
npm run dev
```

访问 http://localhost:3001

### 生产构建

```bash
npm run build
```

### 生产运行

```bash
npm start
```

## 开发指南

### 代码规范

项目使用 ESLint 和 Prettier 进行代码格式化：

```bash
# 检查代码
npm run lint

# 自动修复
npm run lint --fix
```

### 组件开发

所有 UI 组件都遵循以下规范：

1. 使用 TypeScript 编写
2. 使用 Radix UI 作为基础
3. 支持深色主题
4. 完整的类型定义
5. 可访问性支持

示例：

```tsx
import * as React from 'react'
import { Button } from '@/components/ui/button'

export function MyComponent() {
  return (
    <div>
      <Button variant="default">点击我</Button>
    </div>
  )
}
```

### API 调用

使用统一的 API 客户端：

```tsx
import { userApi } from '@/lib/api'

// 获取用户列表
const users = await userApi.list({ page: 1, page_size: 10 })

// 更新用户状态
await userApi.updateStatus(userId, 'active')
```

### 状态管理

使用 Zustand 进行状态管理：

```tsx
import { useAuthStore } from '@/store/auth'

function MyComponent() {
  const { token, logout } = useAuthStore()
  
  return (
    <button onClick={logout}>退出登录</button>
  )
}
```

## 部署

### Docker 部署

项目包含 Dockerfile，可以使用 Docker 部署：

```bash
# 构建镜像
docker build -t nexus-admin .

# 运行容器
docker run -p 3001:3000 nexus-admin
```

### Docker Compose

使用项目根目录的 docker-compose.yml：

```bash
docker compose --profile production up -d
```

### 环境变量

生产环境需要配置以下环境变量：

- `NEXT_PUBLIC_API_URL` - 后端 API 地址
- `NEXT_PUBLIC_APP_URL` - 应用访问地址
- `NEXT_PUBLIC_DEFAULT_TIMEZONE` - 默认时区

## 功能说明

### 仪表盘
- 系统统计数据（用户、渠道、收入、请求）
- 请求趋势图表
- 模型使用分布
- 收入趋势分析
- 系统状态监控

### 用户管理
- 用户列表和搜索
- 用户状态管理
- 余额调整
- 账户启用/禁用

### 渠道管理
- 渠道列表和筛选
- 渠道健康检查
- 渠道测试
- 权重和优先级配置

### 模型管理
- 模型列表和搜索
- 模型定价配置
- 模型启用/禁用
- 功能标签（视觉、函数调用）

### 日志查询
- 请求日志列表
- 多条件筛选
- 统计数据展示
- 日志详情查看

### 订单管理
- 订单列表和筛选
- 订单状态更新
- 订单详情查看
- 支付方式统计

### 资源账户
- 资源账户列表
- 账户状态管理
- 账户刷新
- 错误统计

### 公告管理
- 公告列表和筛选
- 公告发布/归档
- 公告类型和优先级
- 过期时间设置

### 系统设置
- 通用设置（站点名称、URL、时区）
- 安全设置（JWT、会话超时、2FA）
- 通知设置（邮件、Webhook）
- 系统设置（维护模式、注册开关）

## 浏览器支持

- Chrome >= 90
- Firefox >= 88
- Safari >= 14
- Edge >= 90

## 性能优化

- 代码分割和懒加载
- 图片优化
- 字体优化
- 缓存策略
- CDN 支持

## 安全性

- JWT Token 认证
- XSS 防护
- CSRF 防护
- 内容安全策略
- 安全的 HTTP 头

## 故障排除

### 常见问题

**Q: 登录后立即退出？**
A: 检查 JWT Token 是否正确设置，检查后端 API 是否可访问。

**Q: API 请求失败？**
A: 检查 `NEXT_PUBLIC_API_URL` 配置是否正确，检查网络连接。

**Q: 样式不正确？**
A: 确保已安装所有依赖，运行 `npm install`。

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 LICENSE 文件

## 联系方式

- 项目地址: https://github.com/your-org/nexus-admin
- 问题反馈: https://github.com/your-org/nexus-admin/issues
- 邮箱: support@nexus.com

## 更新日志

### v1.0.0 (2026-01-12)
- 🎉 初始版本发布
- ✨ 完成所有核心功能
- 🎨 实现现代化 UI 设计
- 📱 支持响应式布局
- 🔒 添加 JWT 认证
- 📊 实现数据可视化
