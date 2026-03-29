# CPA Core Features Integration

This document describes the CPA (CLIProxyAPI) core features that have been integrated into codex2api.

## Features

### 1. Affinity Scope Isolation

File: `auth/session_affinity.go`

Provides provider-scoped session/affinity key isolation to prevent session conflicts between different providers.

**Key Functions:**
- `auth.GenerateSessionKey(sessionID, provider)` - Generate provider-isolated session key
- `auth.GenerateCacheKey(cacheKey, provider)` - Generate provider-isolated cache key
- `auth.ExtractProviderFromKey(key)` - Extract provider from isolated key
- `auth.IsProviderIsolatedKey(key)` - Check if key has provider isolation

**Usage:**
```go
// Generate provider-isolated session key
sessionKey := auth.GenerateSessionKey("session-123", "codex")
// Result: "codex2api:cdx:session-123"

// Generate cache key for OpenAI provider
cacheKey := auth.GenerateCacheKey("cache-456", "openai")
// Result: "codex2api:oai:cache-456"
```

### 2. Enhanced Proxy Util

File: `proxyutil/proxy.go`

Enhanced proxy configuration with support for direct/none keywords and better environment proxy inheritance.

**Key Features:**
- Support `"direct"` and `"none"` keywords to explicitly bypass proxy
- Environment proxy inheritance via `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`
- `ShouldBypassProxy(targetHost)` - Check if target should bypass proxy based on NO_PROXY

**Usage:**
```go
// Parse proxy setting
setting, err := proxyutil.Parse("direct")
// Mode: ModeDirect - bypass proxy

setting, err := proxyutil.Parse("http://proxy.example.com:8080")
// Mode: ModeProxy - use explicit proxy

// Build HTTP transport
transport, mode, err := proxyutil.BuildHTTPTransport("direct")
// Returns transport with Proxy=nil (bypass)

// Check if target should bypass proxy
shouldBypass := proxyutil.ShouldBypassProxy("api.internal.com")
// Returns true if api.internal.com matches NO_PROXY patterns
```

### 3. Request Diagnostics Logging

File: `proxy/diagnostics.go`

Comprehensive request diagnostics logging inspired by CLIProxyAPI's `logCodexRequestDiagnostics`.

**Key Features:**
- Detailed request metadata logging
- Session continuity tracking
- Authentication info logging
- Reasoning effort/summary extraction

**Configuration:**
```bash
# Enable diagnostics logging
export CODEX_DIAGNOSTICS_ENABLED=true
# or
export DEBUG=true
```

**Usage:**
```go
// Check if diagnostics are enabled
if proxy.IsDiagnosticsEnabled() {
    // Build and log diagnostics
    diag := proxy.BuildRequestDiagnostics(ctx, account, model, body, headers, opts)
    proxy.LogRequestDiagnostics(ctx, diag)
}
```

### 4. Model Alias Resolution

File: `auth/model_alias.go`

Model alias to upstream model resolution with provider-scoped aliases.

**Key Features:**
- Global model aliases
- Provider-specific model aliases (takes priority)
- Suffix preservation (e.g., `-thinking` modifiers)

**Usage:**
```go
// Set global alias
auth.SetGlobalAlias("gpt-4", "gpt-4o")

// Set provider-specific alias
auth.SetProviderAlias("codex", "gpt-4", "gpt-4o-codex")
auth.SetProviderAlias("openai", "gpt-4", "gpt-4o-openai")

// Resolve with provider priority
model := auth.Resolve("gpt-4", "codex")
// Returns "gpt-4o-codex" (provider-specific)

model := auth.Resolve("gpt-4")
// Returns "gpt-4o" (global fallback)

// Resolve with suffix preservation
model := auth.ResolveWithSuffix("gpt-4-thinking", "codex")
// Returns "gpt-4o-thinking" (preserves -thinking suffix)
```

**Built-in Aliases:**
Call `auth.RegisterBuiltinAliases()` to register common aliases:
- `gpt-4` -> `gpt-4o` (codex, openai providers)
- `gpt-3.5` -> `gpt-3.5-turbo`

## Integration Points

### ExecuteRequest Enhancement

The `ExecuteRequest` function in `proxy/executor.go` has been enhanced:

1. **Provider Parameter**: Added optional `provider` parameter for affinity isolation
2. **Session Key Isolation**: Session keys are automatically scoped by provider
3. **Diagnostics Logging**: Automatic diagnostics logging when enabled

**New Signature:**
```go
func ExecuteRequest(ctx context.Context, account *auth.Account, requestBody []byte, sessionID string, proxyOverride string, provider ...string) (*http.Response, error)
```

### Proxy Transport Enhancement

The `ConfigureTransportProxy` function in `auth/proxy_transport.go` now uses `proxyutil`:

- Supports `direct`/`none` keywords
- Better environment proxy handling
- Proper mode-based configuration

## Testing

Run tests for specific packages:

```bash
# Proxy util tests
go test ./proxyutil/...

# Auth tests (session affinity + model alias)
go test ./auth/...

# Proxy diagnostics tests
go test ./proxy/...
```

## Migration Guide

### From Old Proxy Configuration

**Before:**
```go
// Only supported explicit proxy URLs
ConfigureTransportProxy(transport, "http://proxy:8080", dialer)
```

**After:**
```go
// Supports direct/none keywords and environment inheritance
ConfigureTransportProxy(transport, "direct", dialer)     // Bypass proxy
ConfigureTransportProxy(transport, "", dialer)           // Inherit from env
ConfigureTransportProxy(transport, "http://proxy:8080", dialer) // Explicit proxy
```

### From Old Session Resolution

**Before:**
```go
sessionID := ResolveSessionID(authHeader, body)
resp, err := ExecuteRequest(ctx, account, body, sessionID, proxyURL)
```

**After:**
```go
sessionID := ResolveSessionID(authHeader, body)
// Add provider for affinity isolation
resp, err := ExecuteRequest(ctx, account, body, sessionID, proxyURL, "codex")
```

## References

- CLIProxyAPI Reference: `D:\Code\Projects\CLIProxyAPI`
- Original Features:
  - `sdk/proxyutil/proxy.go` - Proxy parsing and transport building
  - `sdk/cliproxy/auth/conductor.go` - Model alias resolution
  - `internal/runtime/executor/codex_continuity.go` - Request diagnostics
