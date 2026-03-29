package auth

import (
	"strings"
	"sync"
)

// ModelAliasResolver 处理模型别名到上游模型名称的映射
// 支持按 provider 隔离的模型别名作用域
type ModelAliasResolver struct {
	mu sync.RWMutex
	// 全局别名映射: alias -> upstream model
	globalAliases map[string]string
	// 按 provider 的别名映射: provider -> alias -> upstream model
	providerAliases map[string]map[string]string
	// 默认 provider 回退链
	defaultChain []string
}

// NewModelAliasResolver 创建新的模型别名解析器
func NewModelAliasResolver() *ModelAliasResolver {
	return &ModelAliasResolver{
		globalAliases:   make(map[string]string),
		providerAliases: make(map[string]map[string]string),
		defaultChain:    []string{"codex", "openai", "anthropic", "gemini"},
	}
}

// SetGlobalAlias 设置全局模型别名
func (r *ModelAliasResolver) SetGlobalAlias(alias, upstreamModel string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	alias = normalizeModelName(alias)
	upstreamModel = strings.TrimSpace(upstreamModel)

	if upstreamModel == "" {
		delete(r.globalAliases, alias)
		return
	}

	r.globalAliases[alias] = upstreamModel
}

// SetProviderAlias 为特定 provider 设置模型别名
func (r *ModelAliasResolver) SetProviderAlias(provider, alias, upstreamModel string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider = strings.ToLower(strings.TrimSpace(provider))
	alias = normalizeModelName(alias)
	upstreamModel = strings.TrimSpace(upstreamModel)

	if provider == "" {
		return
	}

	if r.providerAliases[provider] == nil {
		r.providerAliases[provider] = make(map[string]string)
	}

	if upstreamModel == "" {
		delete(r.providerAliases[provider], alias)
		return
	}

	r.providerAliases[provider][alias] = upstreamModel
}

// Resolve 解析模型别名
// 优先级: provider 特定 > 全局
func (r *ModelAliasResolver) Resolve(model string, providers ...string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model = normalizeModelName(model)

	// 1. 尝试 provider 特定的别名
	for _, provider := range providers {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider == "" {
			continue
		}

		if aliases, ok := r.providerAliases[provider]; ok {
			if upstream, ok := aliases[model]; ok {
				return upstream
			}
		}
	}

	// 2. 尝试全局别名
	if upstream, ok := r.globalAliases[model]; ok {
		return upstream
	}

	// 3. 返回原始模型名（未找到别名映射）
	return model
}

// ResolveWithSuffix 解析模型别名并保留后缀（如 thinking 修饰符）
func (r *ModelAliasResolver) ResolveWithSuffix(model string, providers ...string) string {
	suffix := ""
	baseModel := model

	// 提取后缀（如 -thinking, -reasoning）
	// 只保留包含至少一个字母的后缀（排除纯数字后缀如 -4）
	if idx := strings.LastIndex(model, "-"); idx > 0 {
		potentialSuffix := model[idx:]
		// 检查后缀是否包含字母（功能修饰符通常包含字母）
		hasLetter := false
		for i := 1; i < len(potentialSuffix); i++ {
			if (potentialSuffix[i] >= 'a' && potentialSuffix[i] <= 'z') ||
				(potentialSuffix[i] >= 'A' && potentialSuffix[i] <= 'Z') {
				hasLetter = true
				break
			}
		}
		if hasLetter {
			suffix = potentialSuffix
			baseModel = model[:idx]
		}
	}

	resolved := r.Resolve(baseModel, providers...)
	if resolved == baseModel {
		return model // 没有解析到别名，返回原始模型名
	}

	return resolved + suffix
}

// GetProviderAliases 获取指定 provider 的所有别名
func (r *ModelAliasResolver) GetProviderAliases(provider string) map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return nil
	}

	aliases, ok := r.providerAliases[provider]
	if !ok {
		return nil
	}

	// 返回副本
	result := make(map[string]string, len(aliases))
	for k, v := range aliases {
		result[k] = v
	}
	return result
}

// GetGlobalAliases 获取所有全局别名
func (r *ModelAliasResolver) GetGlobalAliases() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string, len(r.globalAliases))
	for k, v := range r.globalAliases {
		result[k] = v
	}
	return result
}

// ClearProviderAliases 清除指定 provider 的所有别名
func (r *ModelAliasResolver) ClearProviderAliases(provider string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider = strings.ToLower(strings.TrimSpace(provider))
	delete(r.providerAliases, provider)
}

// ClearGlobalAliases 清除所有全局别名
func (r *ModelAliasResolver) ClearGlobalAliases() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.globalAliases = make(map[string]string)
}

// SetDefaultProviderChain 设置默认 provider 回退链
func (r *ModelAliasResolver) SetDefaultProviderChain(chain []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]string, 0, len(chain))
	for _, p := range chain {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	r.defaultChain = filtered
}

// GetDefaultProviderChain 获取默认 provider 回退链
func (r *ModelAliasResolver) GetDefaultProviderChain() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.defaultChain))
	copy(result, r.defaultChain)
	return result
}

// normalizeModelName 规范化模型名称（小写、去空格）
func normalizeModelName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// DefaultResolver 全局默认解析器实例
var DefaultResolver = NewModelAliasResolver()

// SetGlobalAlias 设置全局别名（便捷函数）
func SetGlobalAlias(alias, upstreamModel string) {
	DefaultResolver.SetGlobalAlias(alias, upstreamModel)
}

// SetProviderAlias 设置 provider 别名（便捷函数）
func SetProviderAlias(provider, alias, upstreamModel string) {
	DefaultResolver.SetProviderAlias(provider, alias, upstreamModel)
}

// Resolve 解析模型别名（便捷函数）
func Resolve(model string, providers ...string) string {
	return DefaultResolver.Resolve(model, providers...)
}

// ResolveWithSuffix 解析模型别名并保留后缀（便捷函数）
func ResolveWithSuffix(model string, providers ...string) string {
	return DefaultResolver.ResolveWithSuffix(model, providers...)
}

// RegisterBuiltinAliases 注册内置模型别名
func RegisterBuiltinAliases() {
	// Codex 系列别名映射
	SetProviderAlias("codex", "gpt-4", "gpt-4o")
	SetProviderAlias("codex", "gpt-4-turbo", "gpt-4o")
	SetProviderAlias("codex", "gpt-3.5", "gpt-3.5-turbo")
	SetProviderAlias("codex", "gpt-3.5-turbo", "gpt-3.5-turbo-0125")

	// OpenAI 兼容 provider 的别名
	SetProviderAlias("openai", "gpt-4", "gpt-4o-2024-05-13")
	SetProviderAlias("openai", "gpt-4-turbo", "gpt-4-turbo-2024-04-09")
	SetProviderAlias("openai", "gpt-3.5", "gpt-3.5-turbo-0125")

	// 全局默认别名
	SetGlobalAlias("gpt-4", "gpt-4o")
	SetGlobalAlias("gpt-4-turbo", "gpt-4o")
	SetGlobalAlias("gpt-3.5", "gpt-3.5-turbo")
}
