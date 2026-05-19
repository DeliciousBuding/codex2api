# Codex2API 配置参考

本文档列出了所有可配置项，包括环境变量（物理层）和数据库设置（业务层）。

---

## 环境变量 (`.env`)

### 基础配置

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `CODEX_PORT` | HTTP 服务端口 | `8080` |
| `ADMIN_SECRET` | 管理后台登录密钥（设置后首次访问 `/admin` 会弹出密码框） | 空（不鉴权） |
| `CODEX_API_KEYS` | 下游 API Key 鉴权（逗号分隔，留空则不鉴权） | 空 |
| `CODEX_PROXY_URL` | 全局代理地址（如 `http://host.docker.internal:7890`） | 空 |
| `FAST_SCHEDULER_ENABLED` | 启用快速调度器（`true`） | 空（关闭） |
| `TZ` | 时区 | `Asia/Shanghai` |

### PostgreSQL 模式

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `DATABASE_DRIVER` | 数据库驱动，设为 `postgres` | `postgres` |
| `DATABASE_HOST` | PostgreSQL 主机 | `postgres` |
| `DATABASE_PORT` | PostgreSQL 端口 | `5432` |
| `DATABASE_USER` | PostgreSQL 用户 | — |
| `DATABASE_PASSWORD` | PostgreSQL 密码 | — |
| `DATABASE_NAME` | PostgreSQL 数据库名 | — |
| `DATABASE_SSLMODE` | PostgreSQL SSL 模式 | `disable` |
| `CACHE_DRIVER` | 缓存驱动，设为 `redis` | `redis` |
| `REDIS_ADDR` | Redis 地址 | `redis:6379` |
| `REDIS_PASSWORD` | Redis 密码 | 空 |
| `REDIS_DB` | Redis DB 库号 | `0` |

### SQLite 模式

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `DATABASE_DRIVER` | 数据库驱动，设为 `sqlite` | `sqlite` |
| `DATABASE_PATH` | SQLite 数据文件路径 | `/data/codex2api.db` |
| `CACHE_DRIVER` | 缓存驱动，设为 `memory` | `memory` |

---

## 数据库设置 (管理台修改)

以下设置在管理后台 `/admin/settings` 页面修改，保存在数据库 `system_settings` 表中。

| 设置项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `MaxConcurrency` | int | `2` | 单账号最大并发 |
| `GlobalRPM` | int | `0` | 全局请求速率限制（0 = 不限制） |
| `TestModel` | string | `gpt-5.4` | 测试连接时使用的模型 |
| `TestConcurrency` | int | `50` | 测试连接时的并发数 |
| `ProxyURL` | string | 空 | 全局代理地址（可被账号级覆盖） |
| `PgMaxConns` | int | `50` | PostgreSQL 最大连接数 |
| `RedisPoolSize` | int | `30` | Redis 连接池大小 |
| `AdminSecret` | string | 空 | 管理后台密钥（如果环境变量 `ADMIN_SECRET` 已设置则数据库值不生效） |

### 自动清理开关

| 设置项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `AutoCleanUnauthorized` | bool | `false` | 自动清理 401/banned 账号 |
| `AutoCleanRateLimited` | bool | `false` | 自动清理 429/限流账号 |
| `AutoCleanFullUsage` | bool | `false` | 自动清理用量已满的账号 |
| `AutoCleanError` | bool | `false` | 自动清理 error 状态账号 |

### 代理与调度

| 设置项 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `ProxyPoolEnabled` | bool | `false` | 启用代理池轮询（替代全局单代理） |
| `FastSchedulerEnabled` | bool | `false` | 启用快速调度器（内存调度，热路径轻量化） |
| `SchedulerMode` | string | `round_robin` | 调度模式：`round_robin` 或 `remaining_quota` |
| `MaxRetries` | int | `2` | 透明重试最大次数（0 = 不重试，最大 10） |
| `AllowRemoteMigration` | bool | `false` | 允许远程账号迁移（需先设置 `AdminSecret`） |

### 调度模式详解

- **`round_robin`（默认）**：按健康层级轮询分配，15% 概率随机打散
- **`remaining_quota`**：优先将请求分配给剩余配额最多的账号，适合账号配额差异较大的场景

---

## Credit 配额账号

在账号管理中可以标记特定账号的 credit 属性：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `credit_enabled` | bool | `false` | 标记为 credit 配额账号 |
| `credit_skip_usage_window` | bool | `false` | 跳过用量窗口限制（如 Free 账号的 7 天/5 小时窗口） |

这两个字段可通过管理后台的账号管理或 API 进行修改。设置后调度器不再因用量窗口限制排除该账号。

---

## 代理解析顺序

1. **账号级代理**：`accounts.proxy_url`（每账号独立配置）
2. **全局代理**：`system_settings.proxy_url`（管理台设置，所有账号共用）
3. **系统代理**：环境变量 `HTTPS_PROXY` / `https_proxy` / `HTTP_PROXY` / `http_proxy`（仅当上述两项均为空时生效）
4. **直连**：以上均未配置时直连上游

OAuth 授权流程和 Token 刷新均遵循此解析顺序。
