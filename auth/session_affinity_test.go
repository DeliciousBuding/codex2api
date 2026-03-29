package auth

import (
	"strings"
	"testing"
)

func TestNewSessionAffinityResolver(t *testing.T) {
	r := NewSessionAffinityResolver()
	if r == nil {
		t.Fatal("NewSessionAffinityResolver() returned nil")
	}
	if r.basePrefix == "" {
		t.Error("basePrefix should not be empty")
	}
	if len(r.providerPrefixes) == 0 {
		t.Error("providerPrefixes should not be empty")
	}
}

func TestSetBasePrefix(t *testing.T) {
	r := NewSessionAffinityResolver()

	r.SetBasePrefix("myapp")
	if r.basePrefix != "myapp" {
		t.Errorf("basePrefix = %q, want 'myapp'", r.basePrefix)
	}

	// Empty prefix should default to codex2api
	r.SetBasePrefix("")
	if r.basePrefix != "codex2api" {
		t.Errorf("basePrefix = %q, want 'codex2api'", r.basePrefix)
	}
}

func TestRegisterProviderPrefix(t *testing.T) {
	r := NewSessionAffinityResolver()

	r.RegisterProviderPrefix("custom", "cus")
	if r.providerPrefixes["custom"] != "cus" {
		t.Errorf("providerPrefixes['custom'] = %q, want 'cus'", r.providerPrefixes["custom"])
	}

	// Empty provider should be ignored
	r.RegisterProviderPrefix("", "test")
	if _, ok := r.providerPrefixes[""]; ok {
		t.Error("Empty provider should not be registered")
	}

	// Empty prefix should be ignored
	r.RegisterProviderPrefix("test", "")
	if _, ok := r.providerPrefixes["test"]; ok {
		t.Error("Empty prefix should not be registered")
	}
}

func TestResolveAffinityKey(t *testing.T) {
	r := NewSessionAffinityResolver()

	tests := []struct {
		key      string
		provider string
		expected string
	}{
		{
			key:      "session-123",
			provider: "codex",
			expected: "codex2api:cdx:session-123",
		},
		{
			key:      "session-123",
			provider: "openai",
			expected: "codex2api:oai:session-123",
		},
		{
			key:      "session-123",
			provider: "",
			expected: "codex2api:session-123",
		},
		{
			key:      "",
			provider: "codex",
			expected: "",
		},
		{
			key:      "session-123",
			provider: "UNKNOWN",
			expected: "codex2api:unknown:session-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.key, func(t *testing.T) {
			result := r.ResolveAffinityKey(tt.key, tt.provider)
			if result != tt.expected {
				t.Errorf("ResolveAffinityKey(%q, %q) = %q, want %q",
					tt.key, tt.provider, result, tt.expected)
			}
		})
	}
}

func TestResolveAffinityKeyAlreadyPrefixed(t *testing.T) {
	r := NewSessionAffinityResolver()

	// Key already has the correct prefix
	key := "codex2api:cdx:session-123"
	result := r.ResolveAffinityKey(key, "codex")
	if result != key {
		t.Errorf("ResolveAffinityKey(%q, 'codex') = %q, want %q", key, result, key)
	}

	// Key has different prefix - should be re-prefixed
	key = "codex2api:oai:session-123"
	result = r.ResolveAffinityKey(key, "codex")
	expected := "codex2api:cdx:codex2api:oai:session-123"
	if result != expected {
		t.Errorf("ResolveAffinityKey(%q, 'codex') = %q, want %q", key, result, expected)
	}
}

func TestExtractProviderFromKey(t *testing.T) {
	r := NewSessionAffinityResolver()

	tests := []struct {
		key      string
		expected string
	}{
		{
			key:      "codex2api:cdx:session-123",
			expected: "codex",
		},
		{
			key:      "codex2api:oai:session-123",
			expected: "openai",
		},
		{
			key:      "codex2api:session-123",
			expected: "",
		},
		{
			key:      "invalid-key",
			expected: "",
		},
		{
			key:      "",
			expected: "",
		},
		{
			key:      "codex2api:custom:session-123",
			expected: "custom", // Unknown prefix returns prefix itself
		},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.key, ":", "_"), func(t *testing.T) {
			result := r.ExtractProviderFromKey(tt.key)
			if result != tt.expected {
				t.Errorf("ExtractProviderFromKey(%q) = %q, want %q",
					tt.key, result, tt.expected)
			}
		})
	}
}

func TestIsProviderIsolatedKey(t *testing.T) {
	r := NewSessionAffinityResolver()

	tests := []struct {
		key      string
		expected bool
	}{
		{
			key:      "codex2api:cdx:session-123",
			expected: true,
		},
		{
			key:      "codex2api:session-123",
			expected: false, // No provider prefix
		},
		{
			key:      "session-123",
			expected: false,
		},
		{
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.key, ":", "_"), func(t *testing.T) {
			result := r.IsProviderIsolatedKey(tt.key)
			if result != tt.expected {
				t.Errorf("IsProviderIsolatedKey(%q) = %v, want %v",
					tt.key, result, tt.expected)
			}
		})
	}
}

func TestGenerateSessionKey(t *testing.T) {
	r := NewSessionAffinityResolver()

	result := r.GenerateSessionKey("session-123", "codex")
	expected := "codex2api:cdx:session-123"
	if result != expected {
		t.Errorf("GenerateSessionKey('session-123', 'codex') = %q, want %q", result, expected)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	r := NewSessionAffinityResolver()

	result := r.GenerateCacheKey("cache-456", "openai")
	expected := "codex2api:oai:cache-456"
	if result != expected {
		t.Errorf("GenerateCacheKey('cache-456', 'openai') = %q, want %q", result, expected)
	}
}

func TestSessionAffinityCaseInsensitiveProvider(t *testing.T) {
	r := NewSessionAffinityResolver()

	// Test with different cases
	result := r.ResolveAffinityKey("session-123", "CODEX")
	expected := "codex2api:cdx:session-123"
	if result != expected {
		t.Errorf("ResolveAffinityKey('session-123', 'CODEX') = %q, want %q", result, expected)
	}

	result = r.ResolveAffinityKey("session-123", "Codex")
	if result != expected {
		t.Errorf("ResolveAffinityKey('session-123', 'Codex') = %q, want %q", result, expected)
	}
}

func TestDefaultSessionAffinityResolver(t *testing.T) {
	// Save current state
	oldResolver := DefaultSessionAffinityResolver
	defer func() {
		DefaultSessionAffinityResolver = oldResolver
	}()

	// Create new default resolver for testing
	DefaultSessionAffinityResolver = NewSessionAffinityResolver()

	// Test convenience functions
	result := ResolveAffinityKey("session-123", "codex")
	expected := "codex2api:cdx:session-123"
	if result != expected {
		t.Errorf("ResolveAffinityKey('session-123', 'codex') = %q, want %q", result, expected)
	}

	result = GenerateSessionKey("session-456", "openai")
	expected = "codex2api:oai:session-456"
	if result != expected {
		t.Errorf("GenerateSessionKey('session-456', 'openai') = %q, want %q", result, expected)
	}

	result = ExtractProviderFromKey("codex2api:cdx:session-123")
	if result != "codex" {
		t.Errorf("ExtractProviderFromKey('codex2api:cdx:session-123') = %q, want 'codex'", result)
	}

	isolated := IsProviderIsolatedKey("codex2api:cdx:session-123")
	if !isolated {
		t.Error("IsProviderIsolatedKey('codex2api:cdx:session-123') should be true")
	}
}

func TestAllRegisteredProviders(t *testing.T) {
	r := NewSessionAffinityResolver()

	providers := []string{
		"codex", "openai", "anthropic", "gemini", "vertex",
		"azure", "aws", "cohere", "mistral", "groq",
		"together", "fireworks", "replicate", "perplexity",
	}

	for _, provider := range providers {
		result := r.ResolveAffinityKey("test-session", provider)
		if !strings.HasPrefix(result, "codex2api:") {
			t.Errorf("Provider %q: result %q missing codex2api: prefix", provider, result)
		}
		if !strings.Contains(result, ":") {
			t.Errorf("Provider %q: result %q missing provider separator", provider, result)
		}
	}
}

// BenchmarkResolveAffinityKey benchmarks the ResolveAffinityKey function
func BenchmarkResolveAffinityKey(b *testing.B) {
	r := NewSessionAffinityResolver()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ResolveAffinityKey("session-123", "codex")
	}
}

// BenchmarkExtractProviderFromKey benchmarks the ExtractProviderFromKey function
func BenchmarkExtractProviderFromKey(b *testing.B) {
	r := NewSessionAffinityResolver()
	key := "codex2api:cdx:session-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ExtractProviderFromKey(key)
	}
}
