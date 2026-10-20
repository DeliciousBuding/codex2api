package proxy

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codex2api/auth"
	"github.com/codex2api/cache"
	"github.com/codex2api/database"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

func newSearchPolicyHandler(t *testing.T) (*Handler, *auth.Account) {
	t.Helper()
	db, err := database.New("sqlite", filepath.Join(t.TempDir(), "policy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := auth.NewStore(nil, nil, &database.SystemSettings{MaxConcurrency: 10})
	t.Cleanup(store.Stop)
	account := &auth.Account{DBID: 1, AccessToken: "fixture-token", PlanType: "plus"}
	store.AddAccount(account)
	h := NewHandler(store, nil, nil, nil)
	h.db = db
	runtimeCache := cache.NewMemory(1)
	t.Cleanup(func() { runtimeCache.Close() })
	h.SetRuntimeCache(runtimeCache)
	return h, account
}

func TestAlphaSearchRequestPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var forwarded atomic.Int64
	var responseStatus atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(responseStatus.Load()))
		_, _ = w.Write([]byte(`{"output":[]}`))
	}))
	t.Cleanup(upstream.Close)
	previous := codexAlphaSearchURLForTest
	codexAlphaSearchURLForTest = upstream.URL
	t.Cleanup(func() { codexAlphaSearchURLForTest = previous })
	for _, tc := range []struct {
		name, body, message string
		limits              database.APIKeyLimits
		want                int
		upstreamStatus      int
		prepare             func(*Handler, *database.APIKeyRow) func()
	}{
		{name: "allowed", limits: database.APIKeyLimits{ModelAllow: []string{"gpt-5.5"}}, want: 200},
		{name: "upstream error releases slots", limits: database.APIKeyLimits{MaxConcurrency: 1, ScopeLimits: []database.APIKeyScopeLimit{{ScopeType: database.APIKeyScopeTypeAccount, ScopeID: 1, MaxConcurrency: 1}}}, want: 503, upstreamStatus: 503},
		{name: "model allow", limits: database.APIKeyLimits{ModelAllow: []string{"other"}}, want: 403, message: "not allowed"},
		{name: "model deny", limits: database.APIKeyLimits{ModelDeny: []string{"gpt-5.5"}}, want: 403, message: "denied"},
		{name: "missing restricted model", body: `{"commands":{}}`, limits: database.APIKeyLimits{ModelAllow: []string{"other"}}, want: 403, message: "model"},
		{name: "duplicate model", body: `{"model":"gpt-5.5","mod\u0065l":"other"}`, limits: database.APIKeyLimits{ModelAllow: []string{"gpt-5.5"}}, want: 400, message: "model"},
		{name: "model case alias", body: `{"Model":"gpt-5.5"}`, want: 400, message: "model"},
		{name: "rpm", limits: database.APIKeyLimits{RPM: 1}, want: 429, message: "per minute", prepare: func(h *Handler, row *database.APIKeyRow) func() {
			h.writeAPIKeyLimitCache(context.Background(), apiKeyLimitsCacheKey(row.ID, "req", "rpm"), &database.APIKeyWindowUsage{Requests: 1})
			return func() {}
		}},
		{name: "weekly model requests", limits: database.APIKeyLimits{ModelRequestLimits: []database.APIKeyModelRequestLimit{{Model: "gpt-5.5", MaxRequests: 1}}}, want: 429, message: "weekly model request", prepare: func(h *Handler, row *database.APIKeyRow) func() {
			if hit, err := h.db.ConsumeAPIKeyModelRequest(context.Background(), row.ID, "fixture-precharge", "gpt-5.5", row.Limits.ModelRequestLimits, time.Now()); err != nil || hit != nil {
				t.Fatalf("precharge: hit=%v err=%v", hit, err)
			}
			return func() {}
		}},
		{name: "key concurrency", limits: database.APIKeyLimits{MaxConcurrency: 1}, want: 429, message: "concurrency", prepare: func(h *Handler, row *database.APIKeyRow) func() {
			release, _, ok := h.apiKeyConcurrencyLimiter().acquireTracked(row.ID, 1)
			if !ok {
				t.Fatal("fixture could not acquire key slot")
			}
			return release
		}},
		{name: "scope concurrency", limits: database.APIKeyLimits{ScopeLimits: []database.APIKeyScopeLimit{{ScopeType: database.APIKeyScopeTypeAccount, ScopeID: 1, MaxConcurrency: 1}}}, want: 429, message: "scope concurrency", prepare: func(h *Handler, row *database.APIKeyRow) func() {
			lease := apiKeyScopeConcurrency.acquire(scopeSkipKey{apiKeyID: row.ID, scopeType: database.APIKeyScopeTypeAccount, scopeID: 1})
			return func() { apiKeyScopeConcurrency.release(lease) }
		}},
		{name: "scope fallback", limits: database.APIKeyLimits{ScopeLimits: []database.APIKeyScopeLimit{{ScopeType: database.APIKeyScopeTypeAccount, ScopeID: 1, MaxConcurrency: 1}}}, want: 200, prepare: func(h *Handler, row *database.APIKeyRow) func() {
			h.store.AddAccount(&auth.Account{DBID: 2, AccessToken: "fallback-fixture", PlanType: "plus"})
			lease := apiKeyScopeConcurrency.acquire(scopeSkipKey{apiKeyID: row.ID, scopeType: database.APIKeyScopeTypeAccount, scopeID: 1})
			return func() { apiKeyScopeConcurrency.release(lease) }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, account := newSearchPolicyHandler(t)
			id, err := h.db.InsertAPIKeyWithOptions(context.Background(), database.APIKeyInput{Key: "sk-search-fixture", Name: "search fixture", Limits: tc.limits})
			if err != nil {
				t.Fatal(err)
			}
			row, err := h.db.GetAPIKeyByID(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			if tc.prepare != nil {
				release := tc.prepare(h, row)
				defer release()
			}
			body := tc.body
			if body == "" {
				body = `{"model":"gpt-5.5","commands":{"search_query":[{"q":"fixture"}]}}`
			}
			c, rec := liveTestContext(t, http.MethodPost, "/v1/alpha/search", []byte(body), row)
			before := forwarded.Load()
			keyBefore := h.APIKeyConcurrencySnapshot()[row.ID]
			scopeBefore := APIKeyScopeInflight(row.ID, database.APIKeyScopeTypeAccount, 1)
			status := tc.upstreamStatus
			if status == 0 {
				status = http.StatusOK
			}
			responseStatus.Store(int64(status))
			h.CodexAlphaSearchHandler(c)
			if rec.Code != tc.want || !strings.Contains(rec.Body.String(), tc.message) {
				t.Fatalf("status=%d body=%s, want %d containing %q", rec.Code, rec.Body.String(), tc.want, tc.message)
			}
			if tc.want != 200 && tc.upstreamStatus == 0 && forwarded.Load() != before {
				t.Fatal("rejected request reached upstream")
			}
			if account.ActiveRequests.Load() != 0 {
				t.Fatal("account slot leaked")
			}
			if h.APIKeyConcurrencySnapshot()[row.ID] != keyBefore || APIKeyScopeInflight(row.ID, database.APIKeyScopeTypeAccount, 1) != scopeBefore {
				t.Fatal("request leaked a key or scope concurrency slot")
			}
		})
	}
}

func TestLiveCreateHonorsExplicitModelDeny(t *testing.T) {
	for _, tc := range []struct{ name, session, deny string }{
		{"explicit", `{"model":"gpt-live-1-mini"}`, "gpt-live-1-mini"},
		{"default", `{}`, "gpt-live"},
		{"normalized", `{"model":" GPT-LIVE-1 "}`, "gpt-live-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var generated atomic.Int64
			previous := generateLiveAttestation
			generateLiveAttestation = func(context.Context) (string, error) { generated.Add(1); return "", nil }
			t.Cleanup(func() { generateLiveAttestation = previous })
			h, account := newSearchPolicyHandler(t)
			c, rec := liveTestContext(t, http.MethodPost, "/v1/live", []byte(`{"sdp":"v=0","session":`+tc.session+`}`), &database.APIKeyRow{ID: 81002, Limits: database.APIKeyLimits{AllowLive: true, ModelDeny: []string{tc.deny}}})
			h.LiveCreate(c)
			if rec.Code != 403 || !strings.Contains(rec.Body.String(), "denied") {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if generated.Load() != 0 || account.ActiveRequests.Load() != 0 {
				t.Fatal("denied model caused attestation or account acquisition")
			}
		})
	}
}

func TestWebSocketTurnRevalidatesKey(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		change        func(*testing.T, *Handler, int64)
	}{
		{"disabled", "disabled", func(t *testing.T, h *Handler, id int64) {
			if err := h.db.UpdateAPIKey(context.Background(), id, database.APIKeyUpdate{EnabledSet: true, Enabled: false}); err != nil {
				t.Fatal(err)
			}
		}},
		{"expired", "expired", func(t *testing.T, h *Handler, id int64) {
			if err := h.db.UpdateAPIKeyExpiresAt(context.Background(), id, sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true}); err != nil {
				t.Fatal(err)
			}
		}},
		{"deleted", "Invalid API key", func(t *testing.T, h *Handler, id int64) {
			if err := h.db.DeleteAPIKey(context.Background(), id); err != nil {
				t.Fatal(err)
			}
		}},
		{"quota", "quota exhausted", func(t *testing.T, h *Handler, id int64) {
			if err := h.db.UpdateAPIKeyQuotaLimit(context.Background(), id, 1); err != nil {
				t.Fatal(err)
			}
		}},
		{"model policy", "not allowed", func(t *testing.T, h *Handler, id int64) {
			if err := h.db.UpdateAPIKeyLimits(context.Background(), id, database.APIKeyLimits{ModelAllow: []string{"other"}}); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _, router := newModelQuotaTestHandler(t, 100, "", true)
			// Seed used quota without making any external request.
			id, err := h.db.InsertAPIKeyWithOptions(context.Background(), database.APIKeyInput{Key: "sk-turn-fixture", Name: "turn fixture", QuotaUsed: 2})
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int64
			oldExecute := WebsocketExecuteFunc
			WebsocketExecuteFunc = func(context.Context, *auth.Account, []byte, string, string, string, *DeviceProfileConfig, http.Header, string) (*http.Response, error) {
				calls.Add(1)
				return responsesWSMultiplexResponse("resp_policy"), nil
			}
			t.Cleanup(func() { WebsocketExecuteFunc = oldExecute })
			handled := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(handled)
				router.ServeHTTP(w, r)
			}))
			t.Cleanup(server.Close)
			client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/v1/responses", http.Header{"Authorization": {"Bearer sk-turn-fixture"}})
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				client.Close()
				select {
				case <-handled:
				case <-time.After(5 * time.Second):
					t.Error("websocket handler did not release its workers")
				}
			}()
			responsesWSMultiplexSend(t, client, "", "first")
			responsesWSMultiplexReadTerminal(t, client, "", "resp_policy")
			tc.change(t, h, id)
			responsesWSMultiplexSend(t, client, "", "second")
			for range 16 {
				frame := responsesWSMultiplexRead(t, client)
				if gjson.GetBytes(frame, "type").String() == "error" {
					if !strings.Contains(strings.ToLower(string(frame)), strings.ToLower(tc.message)) {
						t.Fatalf("unexpected rejection: %s", frame)
					}
					if calls.Load() != 1 {
						t.Fatalf("rejected turn reached upstream: calls=%d", calls.Load())
					}
					return
				}
				if gjson.GetBytes(frame, "type").String() == "response.completed" {
					t.Fatalf("changed key was accepted: %s", frame)
				}
			}
			t.Fatal("missing key policy rejection")
		})
	}
}

func TestWebSocketTurnConfigurationReadFailsClosed(t *testing.T) {
	h, row, _ := newModelQuotaTestHandler(t, 100, "", true)
	c, _ := liveTestContext(t, http.MethodGet, "/v1/responses", nil, row)
	ctx, cancel := context.WithCancel(c.Request.Context())
	cancel()
	c.Request = c.Request.WithContext(ctx)
	err := h.refreshAPIKeyWebSocketTurn(c)
	quotaErr := apiKeyModelRequestError(err)
	if quotaErr == nil || quotaErr.status != http.StatusServiceUnavailable {
		t.Fatalf("configuration failure = %v, want 503", err)
	}
}

func TestLiveCallRejectsAmbiguousModel(t *testing.T) {
	for _, session := range []string{
		`{"model":"gpt-live","model":"gpt-live-1-mini"}`,
		`{"model":"gpt-live","mod\u0065l":"gpt-live-1-mini"}`,
		`{"Model":"gpt-live-1-mini"}`, `{"model":42}`,
	} {
		err := ValidateLiveCallRequest(&liveCallRequest{SDP: "v=0", Session: []byte(session)})
		if err == nil {
			t.Fatalf("accepted ambiguous session: %s", session)
		}
	}
}
