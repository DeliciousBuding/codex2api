package auth

import (
	"strings"
	"testing"
)

func TestNewModelAliasResolver(t *testing.T) {
	r := NewModelAliasResolver()
	if r == nil {
		t.Fatal("NewModelAliasResolver() returned nil")
	}
	if r.globalAliases == nil {
		t.Error("globalAliases should not be nil")
	}
	if r.providerAliases == nil {
		t.Error("providerAliases should not be nil")
	}
	if len(r.defaultChain) == 0 {
		t.Error("defaultChain should not be empty")
	}
}

func TestSetAndGetGlobalAlias(t *testing.T) {
	r := NewModelAliasResolver()

	// Test setting alias
	r.SetGlobalAlias("gpt-4", "gpt-4o")

	// Test getting alias
	resolved := r.Resolve("gpt-4")
	if resolved != "gpt-4o" {
		t.Errorf("Resolve('gpt-4') = %q, want 'gpt-4o'", resolved)
	}

	// Test deleting alias (empty upstream model)
	r.SetGlobalAlias("gpt-4", "")
	resolved = r.Resolve("gpt-4")
	if resolved != "gpt-4" {
		t.Errorf("Resolve('gpt-4') after delete = %q, want 'gpt-4'", resolved)
	}
}

func TestSetAndGetProviderAlias(t *testing.T) {
	r := NewModelAliasResolver()

	// Set provider-specific alias
	r.SetProviderAlias("codex", "gpt-4", "gpt-4o-codex")
	r.SetProviderAlias("openai", "gpt-4", "gpt-4o-openai")

	// Resolve with specific provider
	resolved := r.Resolve("gpt-4", "codex")
	if resolved != "gpt-4o-codex" {
		t.Errorf("Resolve('gpt-4', 'codex') = %q, want 'gpt-4o-codex'", resolved)
	}

	resolved = r.Resolve("gpt-4", "openai")
	if resolved != "gpt-4o-openai" {
		t.Errorf("Resolve('gpt-4', 'openai') = %q, want 'gpt-4o-openai'", resolved)
	}

	// Resolve with unknown provider should fall back to global (which is empty)
	resolved = r.Resolve("gpt-4", "unknown")
	if resolved != "gpt-4" {
		t.Errorf("Resolve('gpt-4', 'unknown') = %q, want 'gpt-4'", resolved)
	}
}

func TestProviderAliasPriority(t *testing.T) {
	r := NewModelAliasResolver()

	// Set both global and provider alias
	r.SetGlobalAlias("gpt-4", "gpt-4o-global")
	r.SetProviderAlias("codex", "gpt-4", "gpt-4o-codex")

	// Provider alias should take priority
	resolved := r.Resolve("gpt-4", "codex")
	if resolved != "gpt-4o-codex" {
		t.Errorf("Resolve('gpt-4', 'codex') = %q, want 'gpt-4o-codex'", resolved)
	}

	// Without provider, should use global
	resolved = r.Resolve("gpt-4")
	if resolved != "gpt-4o-global" {
		t.Errorf("Resolve('gpt-4') = %q, want 'gpt-4o-global'", resolved)
	}
}

func TestResolveWithSuffix(t *testing.T) {
	r := NewModelAliasResolver()

	r.SetProviderAlias("codex", "gpt-4", "gpt-4o")

	// Test with suffix
	resolved := r.ResolveWithSuffix("gpt-4-thinking", "codex")
	if resolved != "gpt-4o-thinking" {
		t.Errorf("ResolveWithSuffix('gpt-4-thinking', 'codex') = %q, want 'gpt-4o-thinking'", resolved)
	}

	// Test without suffix
	resolved = r.ResolveWithSuffix("gpt-4", "codex")
	if resolved != "gpt-4o" {
		t.Errorf("ResolveWithSuffix('gpt-4', 'codex') = %q, want 'gpt-4o'", resolved)
	}

	// Test unknown model - should return original
	resolved = r.ResolveWithSuffix("unknown-model", "codex")
	if resolved != "unknown-model" {
		t.Errorf("ResolveWithSuffix('unknown-model', 'codex') = %q, want 'unknown-model'", resolved)
	}
}

func TestGetProviderAliases(t *testing.T) {
	r := NewModelAliasResolver()

	r.SetProviderAlias("codex", "gpt-4", "gpt-4o")
	r.SetProviderAlias("codex", "gpt-3.5", "gpt-3.5-turbo")

	aliases := r.GetProviderAliases("codex")
	if aliases == nil {
		t.Fatal("GetProviderAliases('codex') returned nil")
	}

	if len(aliases) != 2 {
		t.Errorf("len(aliases) = %d, want 2", len(aliases))
	}

	if aliases["gpt-4"] != "gpt-4o" {
		t.Errorf("aliases['gpt-4'] = %q, want 'gpt-4o'", aliases["gpt-4"])
	}

	// Unknown provider should return nil
	aliases = r.GetProviderAliases("unknown")
	if aliases != nil {
		t.Error("GetProviderAliases('unknown') should return nil")
	}
}

func TestClearProviderAliases(t *testing.T) {
	r := NewModelAliasResolver()

	r.SetProviderAlias("codex", "gpt-4", "gpt-4o")
	r.ClearProviderAliases("codex")

	aliases := r.GetProviderAliases("codex")
	if aliases != nil {
		t.Error("GetProviderAliases('codex') should return nil after clear")
	}
}

func TestSetDefaultProviderChain(t *testing.T) {
	r := NewModelAliasResolver()

	chain := []string{"custom1", "custom2", "custom3"}
	r.SetDefaultProviderChain(chain)

	result := r.GetDefaultProviderChain()
	if len(result) != len(chain) {
		t.Errorf("len(chain) = %d, want %d", len(result), len(chain))
	}

	for i, p := range chain {
		if result[i] != p {
			t.Errorf("chain[%d] = %q, want %q", i, result[i], p)
		}
	}
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GPT-4", "gpt-4"},
		{"  gpt-4  ", "gpt-4"},
		{"GPT-4-TURBO", "gpt-4-turbo"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeModelName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeModelName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultResolver(t *testing.T) {
	// Save current state
	oldResolver := DefaultResolver
	defer func() {
		DefaultResolver = oldResolver
	}()

	// Create new default resolver for testing
	DefaultResolver = NewModelAliasResolver()

	// Test convenience functions
	SetGlobalAlias("test-model", "test-upstream")
	resolved := Resolve("test-model")
	if resolved != "test-upstream" {
		t.Errorf("Resolve('test-model') = %q, want 'test-upstream'", resolved)
	}

	SetProviderAlias("codex", "test-model-2", "test-upstream-2")
	resolved = Resolve("test-model-2", "codex")
	if resolved != "test-upstream-2" {
		t.Errorf("Resolve('test-model-2', 'codex') = %q, want 'test-upstream-2'", resolved)
	}
}

func TestRegisterBuiltinAliases(t *testing.T) {
	// Save current state
	oldResolver := DefaultResolver
	defer func() {
		DefaultResolver = oldResolver
	}()

	DefaultResolver = NewModelAliasResolver()
	RegisterBuiltinAliases()

	// Test Codex aliases
	resolved := Resolve("gpt-4", "codex")
	if !strings.Contains(resolved, "gpt-4") {
		t.Errorf("Resolve('gpt-4', 'codex') = %q, should contain 'gpt-4'", resolved)
	}

	// Test global aliases
	resolved = Resolve("gpt-3.5")
	if resolved != "gpt-3.5-turbo" {
		t.Errorf("Resolve('gpt-3.5') = %q, want 'gpt-3.5-turbo'", resolved)
	}
}

func TestCaseInsensitiveProvider(t *testing.T) {
	r := NewModelAliasResolver()

	r.SetProviderAlias("Codex", "gpt-4", "gpt-4o")

	// Should work with different case
	resolved := r.Resolve("gpt-4", "CODEX")
	if resolved != "gpt-4o" {
		t.Errorf("Resolve('gpt-4', 'CODEX') = %q, want 'gpt-4o'", resolved)
	}

	resolved = r.Resolve("gpt-4", "codex")
	if resolved != "gpt-4o" {
		t.Errorf("Resolve('gpt-4', 'codex') = %q, want 'gpt-4o'", resolved)
	}
}

func TestMultipleProvidersFallback(t *testing.T) {
	r := NewModelAliasResolver()

	r.SetProviderAlias("codex", "gpt-4", "gpt-4o-codex")
	r.SetProviderAlias("openai", "gpt-4", "gpt-4o-openai")

	// First provider in list should be checked first
	resolved := r.Resolve("gpt-4", "codex", "openai")
	if resolved != "gpt-4o-codex" {
		t.Errorf("Resolve('gpt-4', 'codex', 'openai') = %q, want 'gpt-4o-codex'", resolved)
	}

	// Reverse order
	resolved = r.Resolve("gpt-4", "openai", "codex")
	if resolved != "gpt-4o-openai" {
		t.Errorf("Resolve('gpt-4', 'openai', 'codex') = %q, want 'gpt-4o-openai'", resolved)
	}

	// Unknown first, should fallback to known
	resolved = r.Resolve("gpt-4", "unknown", "codex")
	if resolved != "gpt-4o-codex" {
		t.Errorf("Resolve('gpt-4', 'unknown', 'codex') = %q, want 'gpt-4o-codex'", resolved)
	}
}

// BenchmarkResolve benchmarks the Resolve function
func BenchmarkResolve(b *testing.B) {
	r := NewModelAliasResolver()
	r.SetProviderAlias("codex", "gpt-4", "gpt-4o")
	r.SetGlobalAlias("gpt-3.5", "gpt-3.5-turbo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Resolve("gpt-4", "codex")
	}
}

// BenchmarkResolveWithSuffix benchmarks the ResolveWithSuffix function
func BenchmarkResolveWithSuffix(b *testing.B) {
	r := NewModelAliasResolver()
	r.SetProviderAlias("codex", "gpt-4", "gpt-4o")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ResolveWithSuffix("gpt-4-thinking", "codex")
	}
}
