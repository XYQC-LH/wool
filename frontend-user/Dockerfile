# 构建阶段
FROM node:18-alpine AS builder

WORKDIR /workspace

ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# 复制 package 文件
COPY frontend-user/package*.json ./frontend-user/

# 安装依赖
WORKDIR /workspace/frontend-user
RUN npm install --no-audit --no-fund

# 复制源代码和 shared-utils
WORKDIR /workspace
COPY packages/shared-utils ./packages/shared-utils
COPY frontend-user ./frontend-user

# 构建应用
WORKDIR /workspace/frontend-user
RUN mkdir -p public && npm run build

# 生产阶段
FROM node:18-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# 创建非 root 用户
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# 复制构建产物
COPY --from=builder /workspace/frontend-user/public ./public
COPY --from=builder /workspace/frontend-user/.next/standalone ./
COPY --from=builder /workspace/frontend-user/.next/static ./.next/static

# 设置正确的权限
USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
