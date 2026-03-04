-- Nexus API Gateway - 数据库初始化脚本
-- 版本: 2.2.0
-- 创建时间: 2026-02-03

-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255),
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    balance DECIMAL(15, 6) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API Token 表
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 渠道表
CREATE TABLE IF NOT EXISTS channels (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    api_key TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 模型表
CREATE TABLE IF NOT EXISTS models (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    pricing_input DECIMAL(10, 6) DEFAULT 0,
    pricing_output DECIMAL(10, 6) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 智能调度核心表

-- 模型能力表
CREATE TABLE IF NOT EXISTS model_capabilities (
    id BIGSERIAL PRIMARY KEY,
    model_id VARCHAR(100) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    operation VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    UNIQUE(model_id, operation)
);

-- 源头组表 (ProviderGroup)
CREATE TABLE IF NOT EXISTS model_providers (
    id SERIAL PRIMARY KEY,
    operation VARCHAR(50) NOT NULL DEFAULT 'chat.completions',
    model_id VARCHAR(100) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    channel_id INT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    actual_cost_per_1k_input DECIMAL(10, 6) NOT NULL DEFAULT 0,
    actual_cost_per_1k_output DECIMAL(10, 6) NOT NULL DEFAULT 0,
    priority INTEGER DEFAULT 0,
    weight INTEGER DEFAULT 1,
    is_cost_priority BOOLEAN DEFAULT true,
    connect_timeout_ms INTEGER DEFAULT 2000,
    attempt_timeout_ms INTEGER DEFAULT 15000,
    stream_first_chunk_timeout_ms INTEGER DEFAULT 3000,
    status VARCHAR(20) DEFAULT 'active',
    failure_threshold INTEGER DEFAULT 3,
    recovery_timeout_seconds INTEGER DEFAULT 30,
    half_open_requests INTEGER DEFAULT 1,
    upstream_model_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(operation, model_id, channel_id)
);

-- 源头实例表 (ProviderInstance)
CREATE TABLE IF NOT EXISTS provider_instances (
    id SERIAL PRIMARY KEY,
    provider_id INT NOT NULL REFERENCES model_providers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    instance_type VARCHAR(20) NOT NULL,
    weight INTEGER DEFAULT 1,
    status VARCHAR(20) DEFAULT 'active',
    max_concurrency INTEGER DEFAULT 0,
    rpm_limit INTEGER DEFAULT 0,
    tpm_limit INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_id, name)
);

-- 生成任务表
CREATE TABLE IF NOT EXISTS generation_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_id UUID REFERENCES tokens(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    model VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    prompt TEXT,
    params JSONB,
    status VARCHAR(20) DEFAULT 'pending',
    progress FLOAT DEFAULT 0,
    result_url TEXT,
    cost DECIMAL(10, 6) DEFAULT 0,
    duration INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- 资源资产表
CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_type VARCHAR(20) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    purpose VARCHAR(50) NOT NULL,
    kind VARCHAR(20) NOT NULL,
    bucket VARCHAR(100) NOT NULL,
    object_key TEXT NOT NULL,
    mime_type VARCHAR(100),
    size_bytes BIGINT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 请求日志表
CREATE TABLE IF NOT EXISTS logs (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    token_id UUID REFERENCES tokens(id) ON DELETE SET NULL,
    model_id VARCHAR(100),
    operation VARCHAR(50),
    provider_id INTEGER,
    status_code INTEGER,
    cost DECIMAL(12, 6) DEFAULT 0,
    latency_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_tokens_user_id ON tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_model_providers_model ON model_providers(model_id);
CREATE INDEX IF NOT EXISTS idx_provider_instances_provider ON provider_instances(provider_id);
CREATE INDEX IF NOT EXISTS idx_generation_tasks_user ON generation_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_generation_tasks_status ON generation_tasks(status);
CREATE INDEX IF NOT EXISTS idx_logs_user ON logs(user_id);
CREATE INDEX IF NOT EXISTS idx_logs_created ON logs(created_at);

