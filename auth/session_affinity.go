package auth

import (
	"fmt"
	"strings"
)

// SessionAffinityResolver 处理 session/affinity key 的作用域隔离
// 支持按 provider 隔离 affinity key，确保不同 provider 的 session 不会冲突
type SessionAffinityResolver struct {
	// 基础前缀，用于命名空间隔离
	basePrefix string
	// provider 特定的前缀映射
	providerPrefixes map[string]string
}

// NewSessionAffinityResolver 创建新的 session affinity 解析器
func NewSessionAffinityResolver() *SessionAffinityResolver {
	return &SessionAffinityResolver{
		basePrefix: "codex2api",
		providerPrefixes: map[string]string{
			"codex":      "cdx",
			"openai":     "oai",
			"anthropic":  "ant",
			"gemini":     "gem",
			"vertex":     "vtx",
			"azure":      "azr",
			"aws":        "aws",
			"cohere":     "chr",
			"mistral":    "mst",
			"groq":       "grq",
			"together":   "tgx",
			"fireworks":  "fwk",
			"replicate":  "rpl",
			"perplexity": "ppx",
		},
	}
}

// SetBasePrefix 设置基础前缀
func (r *SessionAffinityResolver) SetBasePrefix(prefix string) {
	r.basePrefix = strings.TrimSpace(prefix)
	if r.basePrefix == "" {
		r.basePrefix = "codex2api"
	}
}

// RegisterProviderPrefix 注册 provider 前缀
func (r *SessionAffinityResolver) RegisterProviderPrefix(provider, prefix string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	prefix = strings.ToLower(strings.TrimSpace(prefix))

	if provider == "" || prefix == "" {
		return
	}

	r.providerPrefixes[provider] = prefix
}

// ResolveAffinityKey 解析并生成 provider 隔离的 affinity key
// 如果 key 已经包含 provider 前缀，则直接返回
// 否则添加 provider 前缀以确保隔离
func (r *SessionAffinityResolver) ResolveAffinityKey(key, provider string) string {
	if key == "" {
		return ""
	}

	provider = strings.ToLower(strings.TrimSpace(provider))

	// 如果 provider 为空，使用默认前缀
	if provider == "" {
		return fmt.Sprintf("%s:%s", r.basePrefix, key)
	}

	// 获取 provider 前缀
	prefix, ok := r.providerPrefixes[provider]
	if !ok {
		// 未注册的 provider，使用 provider 名本身作为前缀
		prefix = provider
	}

	// 检查 key 是否已经包含该 provider 前缀
	expectedPrefix := fmt.Sprintf("%s:%s:", r.basePrefix, prefix)
	if strings.HasPrefix(key, expectedPrefix) {
		return key // 已经包含前缀，直接返回
	}

	// 生成新的隔离 key
	return fmt.Sprintf("%s:%s:%s", r.basePrefix, prefix, key)
}

// ExtractProviderFromKey 从 affinity key 中提取 provider
func (r *SessionAffinityResolver) ExtractProviderFromKey(key string) string {
	if key == "" {
		return ""
	}

	expectedBase := r.basePrefix + ":"
	if !strings.HasPrefix(key, expectedBase) {
		return ""
	}

	// 移除基础前缀
	trimmed := strings.TrimPrefix(key, expectedBase)

	// 查找第一个冒号
	idx := strings.Index(trimmed, ":")
	if idx < 0 {
		return ""
	}

	prefix := trimmed[:idx]

	// 反向查找 provider
	for provider, p := range r.providerPrefixes {
		if p == prefix {
			return provider
		}
	}

	// 未找到匹配的 provider，返回前缀本身
	return prefix
}

// IsProviderIsolatedKey 检查 key 是否已经按 provider 隔离
func (r *SessionAffinityResolver) IsProviderIsolatedKey(key string) bool {
	if key == "" {
		return false
	}

	expectedBase := r.basePrefix + ":"
	if !strings.HasPrefix(key, expectedBase) {
		return false
	}

	trimmed := strings.TrimPrefix(key, expectedBase)
	return strings.Contains(trimmed, ":")
}

// GenerateSessionKey 生成新的 session key（带 provider 隔离）
func (r *SessionAffinityResolver) GenerateSessionKey(sessionID, provider string) string {
	return r.ResolveAffinityKey(sessionID, provider)
}

// GenerateCacheKey 生成缓存 key（带 provider 隔离）
func (r *SessionAffinityResolver) GenerateCacheKey(cacheKey, provider string) string {
	return r.ResolveAffinityKey(cacheKey, provider)
}

// DefaultSessionAffinityResolver 全局默认 session affinity 解析器
var DefaultSessionAffinityResolver = NewSessionAffinityResolver()

// ResolveAffinityKey 解析 affinity key（便捷函数）
func ResolveAffinityKey(key, provider string) string {
	return DefaultSessionAffinityResolver.ResolveAffinityKey(key, provider)
}

// GenerateSessionKey 生成 session key（便捷函数）
func GenerateSessionKey(sessionID, provider string) string {
	return DefaultSessionAffinityResolver.GenerateSessionKey(sessionID, provider)
}

// GenerateCacheKey 生成缓存 key（便捷函数）
func GenerateCacheKey(cacheKey, provider string) string {
	return DefaultSessionAffinityResolver.GenerateCacheKey(cacheKey, provider)
}

// ExtractProviderFromKey 从 key 中提取 provider（便捷函数）
func ExtractProviderFromKey(key string) string {
	return DefaultSessionAffinityResolver.ExtractProviderFromKey(key)
}

// IsProviderIsolatedKey 检查 key 是否已隔离（便捷函数）
func IsProviderIsolatedKey(key string) bool {
	return DefaultSessionAffinityResolver.IsProviderIsolatedKey(key)
}
