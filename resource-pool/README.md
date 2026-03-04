# 资源池（Python 引擎服务）

该目录用于承载“逆向/不稳定上游”的 Python 适配器与运行时服务，属于系统架构中的 **Resource Pool（引擎）** 层。

## 本地运行（Docker Compose）

该服务默认通过 `docker-compose.yml` 启动（服务名：`resource-pool`），用于承载如 `banana/seedream/sora2` 等适配器。

### 关键环境变量

- `SORA2_API_HOST` / `SORA2_API_KEY`
- `BANANA_API_HOST` / `BANANA_API_KEY`
- `SEEDREAM_BASE_URL` / `SEEDREAM_API_KEY`
- `JIMENG_POOL_DB_PATH` / `JIMENG_BASE_CONFIG_JSON`（Jimeng 号池，可选）

### 存储（与 Go 后端对齐）

- `STORAGE_DRIVER=local|oss`
- `STORAGE_LOCAL_DIR`（local 模式写入目录）
- `STORAGE_SIGN_SECRET`（local 模式签名密钥；为空时默认使用 `JWT_SECRET`）
- `OSS_*`（oss 模式下使用）

本地模式下，该服务会把产物写入与 Go 后端共享的卷目录，并返回 `GET /objects/...` 的签名 URL（由 Go 后端提供下载入口）。

## Jimeng 号池接口

> 说明：为了避免泄露，账号 `sessionid` 在查询接口中会被掩码处理。

- `GET /v1/jimeng/accounts`（账号列表）
- `POST /v1/jimeng/accounts`（新增账号）
- `PATCH /v1/jimeng/accounts/{account_id}`（更新/启用/禁用）
- `POST /v1/jimeng/pool/reload`（手动 reload；若号池忙碌会返回 409）
- `GET /v1/jimeng/pool/accounts`（查看运行时状态/熔断/缓存积分）
- `POST /v1/jimeng/accounts/{account_id}/credit`（刷新积分）
- `POST /v1/jimeng/accounts/{account_id}/daily-credit`（领取每日积分）
