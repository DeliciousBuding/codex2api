package proxy

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/codex2api/auth"
	"github.com/tidwall/gjson"
)

// RequestDiagnostics 请求诊断信息
type RequestDiagnostics struct {
	// 请求标识
	RequestID      string
	AccountID      int64
	Provider       string
	Model          string
	SessionID      string

	// 认证信息
	AuthType       string // "oauth", "api_key"
	AuthID         string
	AuthFile       string
	SelectedAuthID string

	// 会话连续性
	ContinuitySource string // "prompt_cache_key", "execution_session", "idempotency_key", "client_principal", "auth_id"
	SessionKey       string

	// 请求元数据
	Store                bool
	HasInstructions      bool
	ReasoningEffort      string
	ReasoningSummary     string
	PromptCacheKey       string
	PromptCacheRetention string
	ChatGPTAccountID     bool
	Originator           string

	// 格式信息
	SourceFormat string

	// 时间戳
	Timestamp time.Time
}

// DiagnosticsLogger 诊断日志记录器
type DiagnosticsLogger struct {
	mu       sync.RWMutex
	enabled  bool
	debugLog bool
}

// DefaultDiagnosticsLogger 全局默认诊断日志记录器
var DefaultDiagnosticsLogger = NewDiagnosticsLogger()

// NewDiagnosticsLogger 创建新的诊断日志记录器
func NewDiagnosticsLogger() *DiagnosticsLogger {
	return &DiagnosticsLogger{
		enabled:  os.Getenv("CODEX_DIAGNOSTICS_ENABLED") != "" || os.Getenv("DEBUG") != "",
		debugLog: os.Getenv("DEBUG") != "" || os.Getenv("CODEX_DEBUG") != "",
	}
}

// IsEnabled 检查诊断日志是否启用
func (d *DiagnosticsLogger) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEnabled 设置诊断日志启用状态
func (d *DiagnosticsLogger) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// IsDebugEnabled 检查调试日志是否启用
func (d *DiagnosticsLogger) IsDebugEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.debugLog
}

// SetDebugEnabled 设置调试日志启用状态
func (d *DiagnosticsLogger) SetDebugEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.debugLog = enabled
}

// Log 记录诊断日志
func (d *DiagnosticsLogger) Log(ctx context.Context, diag *RequestDiagnostics) {
	if !d.IsEnabled() && !d.IsDebugEnabled() {
		return
	}

	if diag == nil {
		return
	}

	// 构建诊断日志消息（参考 CLIProxyAPI logCodexRequestDiagnostics 格式）
	msg := fmt.Sprintf(
		"[codex request diagnostics] request_id=%s auth_id=%s selected_auth_id=%s auth_file=%s provider=%s model=%s exec_session=%s continuity_source=%s session_id=%s prompt_cache_key=%s prompt_cache_retention=%s store=%t has_instructions=%t reasoning_effort=%s reasoning_summary=%s chatgpt_account_id=%t originator=%s source_format=%s",
		diag.RequestID,
		diag.AuthID,
		diag.SelectedAuthID,
		diag.AuthFile,
		diag.Provider,
		diag.Model,
		diag.SessionID,
		diag.ContinuitySource,
		diag.SessionKey,
		diag.PromptCacheKey,
		diag.PromptCacheRetention,
		diag.Store,
		diag.HasInstructions,
		diag.ReasoningEffort,
		diag.ReasoningSummary,
		diag.ChatGPTAccountID,
		diag.Originator,
		diag.SourceFormat,
	)

	log.Println(msg)
}

// LogRequest 记录请求诊断日志
func (d *DiagnosticsLogger) LogRequest(ctx context.Context, account *auth.Account, model, sessionID string, body []byte, headers http.Header) {
	if !d.IsEnabled() && !d.IsDebugEnabled() {
		return
	}

	diag := &RequestDiagnostics{
		Timestamp:    time.Now(),
		Model:        model,
		SessionID:    sessionID,
		Originator:   headers.Get("Originator"),
		SessionKey:   headers.Get("Session_id"),
		RequestID:    getRequestID(ctx),
	}

	if account != nil {
		account.Mu().RLock()
		diag.AccountID = account.DBID
		account.Mu().RUnlock()
	}

	// 从请求体提取诊断信息
	if len(body) > 0 {
		diag.PromptCacheKey = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
		diag.PromptCacheRetention = gjson.GetBytes(body, "prompt_cache_retention").String()
		diag.Store = gjson.GetBytes(body, "store").Bool()
		diag.HasInstructions = gjson.GetBytes(body, "instructions").Exists()
		diag.ReasoningEffort = gjson.GetBytes(body, "reasoning.effort").String()
		diag.ReasoningSummary = gjson.GetBytes(body, "reasoning.summary").String()
	}

	// 从 headers 提取信息
	if headers != nil {
		diag.ChatGPTAccountID = strings.TrimSpace(headers.Get("Chatgpt-Account-Id")) != ""
	}

	d.Log(ctx, diag)
}

// ExtractContinuitySource 提取会话连续性来源
func ExtractContinuitySource(body []byte, headers http.Header) (source, key string) {
	// 1. 优先从 body 的 prompt_cache_key 提取
	if key = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); key != "" {
		return "prompt_cache_key", key
	}

	// 2. 从 headers 的 session_id 提取
	if key = strings.TrimSpace(headers.Get("Session_id")); key != "" {
		return "session_id", key
	}

	// 3. 从 headers 的 conversation_id 提取
	if key = strings.TrimSpace(headers.Get("Conversation_id")); key != "" {
		return "conversation_id", key
	}

	// 4. 从 headers 的 idempotency_key 提取
	if key = strings.TrimSpace(headers.Get("Idempotency-Key")); key != "" {
		return "idempotency_key", key
	}

	return "", ""
}

// BuildRequestDiagnostics 构建请求诊断信息
func BuildRequestDiagnostics(
	ctx context.Context,
	account *auth.Account,
	model string,
	body []byte,
	headers http.Header,
	opts ExecuteOptions,
) *RequestDiagnostics {
	diag := &RequestDiagnostics{
		Timestamp: time.Now(),
		Model:     model,
		RequestID: getRequestID(ctx),
	}

	if account != nil {
		account.Mu().RLock()
		diag.AccountID = account.DBID
		diag.AuthID = fmt.Sprintf("%d", account.DBID)
		account.Mu().RUnlock()
	}

	// 提取连续性信息
	source, key := ExtractContinuitySource(body, headers)
	diag.ContinuitySource = source
	diag.SessionKey = key

	// 从请求体提取诊断信息
	if len(body) > 0 {
		diag.PromptCacheKey = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
		diag.PromptCacheRetention = gjson.GetBytes(body, "prompt_cache_retention").String()
		diag.Store = gjson.GetBytes(body, "store").Bool()
		diag.HasInstructions = gjson.GetBytes(body, "instructions").Exists()
		diag.ReasoningEffort = gjson.GetBytes(body, "reasoning.effort").String()
		diag.ReasoningSummary = gjson.GetBytes(body, "reasoning.summary").String()
	}

	// 从 headers 提取信息
	if headers != nil {
		diag.ChatGPTAccountID = strings.TrimSpace(headers.Get("Chatgpt-Account-Id")) != ""
		diag.Originator = headers.Get("Originator")
		diag.SessionID = headers.Get("Session_id")
	}

	// 从 options 提取信息
	if opts.SourceFormat != "" {
		diag.SourceFormat = opts.SourceFormat
	}

	return diag
}

// ExecuteOptions 执行选项
type ExecuteOptions struct {
	SourceFormat string
	Metadata     map[string]any
}

// LogRequestDiagnostics 记录请求诊断日志（便捷函数）
func LogRequestDiagnostics(ctx context.Context, diag *RequestDiagnostics) {
	DefaultDiagnosticsLogger.Log(ctx, diag)
}

// SetDiagnosticsEnabled 设置诊断日志启用状态（便捷函数）
func SetDiagnosticsEnabled(enabled bool) {
	DefaultDiagnosticsLogger.SetEnabled(enabled)
}

// SetDebugEnabled 设置调试日志启用状态（便捷函数）
func SetDebugEnabled(enabled bool) {
	DefaultDiagnosticsLogger.SetDebugEnabled(enabled)
}

// IsDiagnosticsEnabled 检查诊断日志是否启用（便捷函数）
func IsDiagnosticsEnabled() bool {
	return DefaultDiagnosticsLogger.IsEnabled()
}

// IsDebugEnabled 检查调试日志是否启用（便捷函数）
func IsDebugEnabled() bool {
	return DefaultDiagnosticsLogger.IsDebugEnabled()
}

// getRequestID 从 context 获取 request ID
func getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// 尝试从 context 获取 request_id
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// formatBytes 格式化字节数组为字符串（用于日志）
func formatBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return string(bytes.TrimSpace(b))
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
