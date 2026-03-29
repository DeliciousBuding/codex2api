package proxy

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/codex2api/auth"
)

func TestNewDiagnosticsLogger(t *testing.T) {
	d := NewDiagnosticsLogger()
	if d == nil {
		t.Fatal("NewDiagnosticsLogger() returned nil")
	}
}

func TestDiagnosticsLogger_SetEnabled(t *testing.T) {
	d := NewDiagnosticsLogger()

	// Test setting enabled
	d.SetEnabled(true)
	if !d.IsEnabled() {
		t.Error("IsEnabled() should be true after SetEnabled(true)")
	}

	d.SetEnabled(false)
	if d.IsEnabled() {
		t.Error("IsEnabled() should be false after SetEnabled(false)")
	}
}

func TestDiagnosticsLogger_SetDebugEnabled(t *testing.T) {
	d := NewDiagnosticsLogger()

	// Test setting debug enabled
	d.SetDebugEnabled(true)
	if !d.IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be true after SetDebugEnabled(true)")
	}

	d.SetDebugEnabled(false)
	if d.IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be false after SetDebugEnabled(false)")
	}
}

func TestExtractContinuitySource(t *testing.T) {
	tests := []struct {
		name       string
		body       []byte
		headers    http.Header
		wantSource string
		wantKey    string
	}{
		{
			name:       "empty",
			body:       []byte("{}"),
			headers:    nil,
			wantSource: "",
			wantKey:    "",
		},
		{
			name:       "prompt_cache_key in body",
			body:       []byte(`{"prompt_cache_key": "cache-123"}`),
			headers:    nil,
			wantSource: "prompt_cache_key",
			wantKey:    "cache-123",
		},
		{
			name:       "session_id in header",
			body:       []byte("{}"),
			headers:    http.Header{"Session_id": []string{"session-456"}},
			wantSource: "session_id",
			wantKey:    "session-456",
		},
		{
			name:       "conversation_id in header",
			body:       []byte("{}"),
			headers:    http.Header{"Conversation_id": []string{"conv-789"}},
			wantSource: "conversation_id",
			wantKey:    "conv-789",
		},
		{
			name:       "idempotency_key in header",
			body:       []byte("{}"),
			headers:    http.Header{"Idempotency-Key": []string{"idem-abc"}},
			wantSource: "idempotency_key",
			wantKey:    "idem-abc",
		},
		{
			name:       "prompt_cache_key priority",
			body:       []byte(`{"prompt_cache_key": "cache-123"}`),
			headers:    http.Header{"Session_id": []string{"session-456"}},
			wantSource: "prompt_cache_key",
			wantKey:    "cache-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, key := ExtractContinuitySource(tt.body, tt.headers)
			if source != tt.wantSource {
				t.Errorf("ExtractContinuitySource() source = %q, want %q", source, tt.wantSource)
			}
			if key != tt.wantKey {
				t.Errorf("ExtractContinuitySource() key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestBuildRequestDiagnostics(t *testing.T) {
	body := []byte(`{
		"prompt_cache_key": "cache-123",
		"prompt_cache_retention": "7d",
		"store": true,
		"instructions": "Test instructions",
		"reasoning": {
			"effort": "high",
			"summary": "verbose"
		}
	}`)

	headers := http.Header{
		"Session_id":         []string{"session-456"},
		"Chatgpt-Account-Id": []string{"account-789"},
		"Originator":         []string{"codex_cli"},
	}

	opts := ExecuteOptions{
		SourceFormat: "openai",
	}

	diag := BuildRequestDiagnostics(context.Background(), nil, "gpt-4", body, headers, opts)

	if diag == nil {
		t.Fatal("BuildRequestDiagnostics() returned nil")
	}

	if diag.Model != "gpt-4" {
		t.Errorf("Model = %q, want 'gpt-4'", diag.Model)
	}

	if diag.PromptCacheKey != "cache-123" {
		t.Errorf("PromptCacheKey = %q, want 'cache-123'", diag.PromptCacheKey)
	}

	if diag.PromptCacheRetention != "7d" {
		t.Errorf("PromptCacheRetention = %q, want '7d'", diag.PromptCacheRetention)
	}

	if !diag.Store {
		t.Error("Store should be true")
	}

	if !diag.HasInstructions {
		t.Error("HasInstructions should be true")
	}

	if diag.ReasoningEffort != "high" {
		t.Errorf("ReasoningEffort = %q, want 'high'", diag.ReasoningEffort)
	}

	if diag.ReasoningSummary != "verbose" {
		t.Errorf("ReasoningSummary = %q, want 'verbose'", diag.ReasoningSummary)
	}

	if !diag.ChatGPTAccountID {
		t.Error("ChatGPTAccountID should be true")
	}

	if diag.Originator != "codex_cli" {
		t.Errorf("Originator = %q, want 'codex_cli'", diag.Originator)
	}

	if diag.SessionKey != "cache-123" {
		t.Errorf("SessionKey = %q, want 'cache-123'", diag.SessionKey)
	}

	if diag.SourceFormat != "openai" {
		t.Errorf("SourceFormat = %q, want 'openai'", diag.SourceFormat)
	}

	if diag.ContinuitySource != "prompt_cache_key" {
		t.Errorf("ContinuitySource = %q, want 'prompt_cache_key'", diag.ContinuitySource)
	}
}

func TestBuildRequestDiagnostics_WithAccount(t *testing.T) {
	account := &auth.Account{}
	// 使用反射或直接访问来设置字段
	// 注意：这里假设我们可以访问 DBID 字段
	// 实际上可能需要使用其他方式设置

	body := []byte(`{}`)
	headers := http.Header{}
	opts := ExecuteOptions{}

	diag := BuildRequestDiagnostics(context.Background(), account, "gpt-4", body, headers, opts)

	if diag == nil {
		t.Fatal("BuildRequestDiagnostics() returned nil")
	}

	// AccountID 应该从 account 提取
	if diag.AccountID != 0 {
		t.Errorf("AccountID = %d, want 0 (not set)", diag.AccountID)
	}
}

func TestRequestDiagnostics_LogOutput(t *testing.T) {
	diag := &RequestDiagnostics{
		Timestamp:            time.Now(),
		RequestID:            "req-123",
		AccountID:            456,
		Provider:             "codex",
		Model:                "gpt-4",
		SessionID:            "session-789",
		AuthType:             "oauth",
		AuthID:               "auth-abc",
		AuthFile:             "auth.json",
		SelectedAuthID:       "selected-def",
		ContinuitySource:     "prompt_cache_key",
		SessionKey:           "cache-ghi",
		Store:                true,
		HasInstructions:      true,
		ReasoningEffort:      "high",
		ReasoningSummary:     "verbose",
		PromptCacheKey:       "cache-ghi",
		PromptCacheRetention: "7d",
		ChatGPTAccountID:     true,
		Originator:           "codex_cli",
		SourceFormat:         "openai",
	}

	// 验证诊断信息字段
	if diag.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want 'req-123'", diag.RequestID)
	}

	if diag.Provider != "codex" {
		t.Errorf("Provider = %q, want 'codex'", diag.Provider)
	}
}

func TestGetRequestID(t *testing.T) {
	// Test with nil context
	id := getRequestID(nil)
	if id != "" {
		t.Errorf("getRequestID(nil) = %q, want ''", id)
	}

	// Test with context without request_id
	ctx := context.Background()
	id = getRequestID(ctx)
	if id != "" {
		t.Errorf("getRequestID(ctx) = %q, want ''", id)
	}

	// Test with context with request_id
	ctx = context.WithValue(context.Background(), "request_id", "req-123")
	id = getRequestID(ctx)
	if id != "req-123" {
		t.Errorf("getRequestID(ctx with request_id) = %q, want 'req-123'", id)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte{}, ""},
		{[]byte("hello"), "hello"},
		{[]byte(" hello "), "hello"},
		{[]byte(" hello\n\t "), "hello"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s       string
		maxLen  int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		result := truncateString(tt.s, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, result, tt.expected)
		}
	}
}

func TestIsDiagnosticsEnabled(t *testing.T) {
	// Save current state
	oldLogger := DefaultDiagnosticsLogger
	defer func() {
		DefaultDiagnosticsLogger = oldLogger
	}()

	DefaultDiagnosticsLogger = NewDiagnosticsLogger()

	// Test default state
	enabled := IsDiagnosticsEnabled()
	// 结果取决于环境变量，无法断言具体值
	_ = enabled

	// Test after setting
	SetDiagnosticsEnabled(true)
	if !IsDiagnosticsEnabled() {
		t.Error("IsDiagnosticsEnabled() should be true after SetDiagnosticsEnabled(true)")
	}

	SetDiagnosticsEnabled(false)
	if IsDiagnosticsEnabled() {
		t.Error("IsDiagnosticsEnabled() should be false after SetDiagnosticsEnabled(false)")
	}
}

func TestIsDebugEnabled(t *testing.T) {
	// Save current state
	oldLogger := DefaultDiagnosticsLogger
	defer func() {
		DefaultDiagnosticsLogger = oldLogger
	}()

	DefaultDiagnosticsLogger = NewDiagnosticsLogger()

	// Test after setting
	SetDebugEnabled(true)
	if !IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be true after SetDebugEnabled(true)")
	}

	SetDebugEnabled(false)
	if IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be false after SetDebugEnabled(false)")
	}
}

func TestDiagnosticsLogger_LogWithNilDiag(t *testing.T) {
	d := NewDiagnosticsLogger()
	d.SetEnabled(true)

	// Should not panic with nil diag
	d.Log(context.Background(), nil)
}

func TestDiagnosticsLogger_LogWhenDisabled(t *testing.T) {
	d := NewDiagnosticsLogger()
	d.SetEnabled(false)
	d.SetDebugEnabled(false)

	diag := &RequestDiagnostics{
		RequestID: "test",
		Model:     "gpt-4",
	}

	// Should not panic when disabled
	d.Log(context.Background(), diag)
}

func BenchmarkExtractContinuitySource(b *testing.B) {
	body := []byte(`{"prompt_cache_key": "cache-123"}`)
	headers := http.Header{
		"Session_id": []string{"session-456"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractContinuitySource(body, headers)
	}
}

func BenchmarkBuildRequestDiagnostics(b *testing.B) {
	body := []byte(`{
		"prompt_cache_key": "cache-123",
		"store": true,
		"reasoning": {"effort": "high"}
	}`)
	headers := http.Header{
		"Session_id": []string{"session-456"},
	}
	opts := ExecuteOptions{SourceFormat: "openai"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildRequestDiagnostics(ctx, nil, "gpt-4", body, headers, opts)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
