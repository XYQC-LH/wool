# Build stage
FROM node:18-alpine AS builder

WORKDIR /workspace

ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# Copy package files
COPY frontend-admin/package*.json ./frontend-admin/

# Install dependencies
WORKDIR /workspace/frontend-admin
RUN npm install --no-audit --no-fund

# Copy source code and shared-utils
WORKDIR /workspace
COPY packages/shared-utils ./packages/shared-utils
COPY frontend-admin ./frontend-admin

# Build the application
WORKDIR /workspace/frontend-admin
RUN mkdir -p public && npm run build

# Production stage
FROM node:18-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder /workspace/frontend-admin/public ./public
COPY --from=builder /workspace/frontend-admin/.next/standalone ./
COPY --from=builder /workspace/frontend-admin/.next/static ./.next/static

# Set correct permissions
USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
