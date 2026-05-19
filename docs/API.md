# Codex2API 管理 API 参考

所有管理 API 位于 `/api/admin/` 路径下，需携带 `X-Admin-Key` 请求头进行鉴权。

---

## 系统设置

### 获取设置

```http
GET /api/admin/settings
```

响应包含所有系统设置项，包括 `scheduler_mode`、`max_retries` 等：

```json
{
  "max_concurrency": 2,
  "global_rpm": 0,
  "test_model": "gpt-5.4",
  "test_concurrency": 50,
  "proxy_url": "",
  "pg_max_conns": 50,
  "redis_pool_size": 30,
  "auto_clean_unauthorized": false,
  "auto_clean_rate_limited": false,
  "auto_clean_full_usage": false,
  "auto_clean_error": false,
  "proxy_pool_enabled": false,
  "fast_scheduler_enabled": false,
  "max_retries": 2,
  "scheduler_mode": "round_robin",
  "admin_secret": "",
  "admin_auth_source": "disabled",
  "allow_remote_migration": false,
  "database_driver": "postgres",
  "database_label": "PostgreSQL",
  "cache_driver": "redis",
  "cache_label": "Redis"
}
```

### 更新设置

```http
PUT /api/admin/settings
```

请求体（所有字段可选，仅提交需要修改的字段）：

```json
{
  "max_concurrency": 4,
  "scheduler_mode": "remaining_quota",
  "max_retries": 3
}
```

支持更新的字段列表见 [CONFIGURATION.md](./CONFIGURATION.md)。

`SchedulerMode` 可选值：
- `round_robin` — 按健康层级轮询分配（默认）
- `remaining_quota` — 优先分配给剩余配额最多的账号

---

## 账号管理

### 获取账号列表

```http
GET /api/admin/accounts
```

返回所有账号，包含 credit 标记字段：

```json
[
  {
    "id": 1,
    "name": "account-1",
    "status": "active",
    "credit_enabled": false,
    "credit_skip_usage_window": false,
    ...
  }
]
```

### 账号 Credit 设置

**PUT** `/api/admin/accounts/:id/credit`

更新指定账号的 credit 配额标记。

请求体：

```json
{
  "credit_enabled": true,
  "credit_skip_usage_window": true
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `credit_enabled` | bool | 是否标记为 credit 配额账号 |
| `credit_skip_usage_window` | bool | 是否跳过用量窗口限制 |

成功响应：

```json
{
  "id": 1,
  "credit_enabled": true,
  "credit_skip_usage_window": true,
  "updated_at": "2026-05-19T12:00:00Z"
}
```

### 获取账号用量

```http
GET /api/admin/accounts/:id/usage
```

返回该账号的 Token 用量统计与模型分布。

---

## API Key 管理

### 获取 API Keys

```http
GET /api/admin/keys
```

### 创建 API Key

```http
POST /api/admin/keys
```

```json
{
  "name": "customer-a",
  "key": "sk-my-custom-key"
}
```

### 删除 API Key

```http
DELETE /api/admin/keys/:id
```

---

## 用量统计

### 获取用量统计

```http
GET /api/admin/usage/stats
```

### 获取用量日志

```http
GET /api/admin/usage/logs?page=1&page_size=20
```

每条日志包含 `api_key_id` 字段，用于按 API Key 追踪用量。

### 清空用量日志

```http
DELETE /api/admin/usage/logs
```

清空前会先将累计值快照到基线表。

---

## 代理池管理

### 获取代理列表

```http
GET /api/admin/proxies
```

### 添加代理

```http
POST /api/admin/proxies
```

```json
{
  "urls": ["http://proxy1:8080", "http://proxy2:8080"],
  "label": "my-proxies"
}
```

### 测试代理

```http
POST /api/admin/proxies/test
```

```json
{
  "url": "http://proxy1:8080"
}
```

---

## OAuth 授权

### 生成授权 URL

```http
POST /api/admin/oauth/generate-auth-url
```

```json
{
  "proxy_url": "http://proxy:7890",
  "redirect_uri": "http://localhost:1455/auth/callback"
}
```

### 兑换授权码

```http
POST /api/admin/oauth/exchange-code
```

```json
{
  "session_id": "...",
  "code": "...",
  "state": "...",
  "name": "my-account",
  "proxy_url": ""
}
```

---

## 运维接口

### 健康检查（公开，无需鉴权）

```http
GET /health
```

### 运维概览

```http
GET /api/admin/ops/overview
```

### 模型列表

```http
GET /api/admin/models
```

### 仪表盘统计

```http
GET /api/admin/stats
```
