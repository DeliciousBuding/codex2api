package auth

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/codex2api/database"
)

func TestLoadFromDBRestoresModelStatesAndAggregateCooldown(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "codex2api.db")

	db, err := database.New("sqlite", dbPath)
	if err != nil {
		t.Fatalf("database.New(sqlite) 返回错误: %v", err)
	}
	defer db.Close()

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open(sqlite) 返回错误: %v", err)
	}
	defer sqlDB.Close()

	now := time.Now().UTC()
	later := now.Add(30 * time.Minute)
	credentials := `{"access_token":"token-1","email":"test@example.com","plan_type":"team"}`
	modelStates := `{"gpt-5":{"status":1,"unavailable":true,"next_retry_after":"` + later.Format(time.RFC3339) + `","last_error":"rate_limited","strike_count":1,"backoff_level":0,"updated_at":"` + now.Format(time.RFC3339) + `"}}`

	_, err = sqlDB.ExecContext(context.Background(), `
		INSERT INTO accounts (name, credentials, proxy_url, status, cooldown_reason, cooldown_until, model_states)
		VALUES (?, ?, ?, 'active', '', NULL, ?)
	`, "acc-1", credentials, "", modelStates)
	if err != nil {
		t.Fatalf("插入测试账号失败: %v", err)
	}

	store := &Store{
		db:           db,
		maxConcurrency: 2,
	}

	if err := store.loadFromDB(context.Background()); err != nil {
		t.Fatalf("loadFromDB() 返回错误: %v", err)
	}

	if len(store.accounts) != 1 {
		t.Fatalf("loadFromDB() 加载账号数 = %d, want 1", len(store.accounts))
	}

	acc := store.accounts[0]
	if acc.ModelStates == nil {
		t.Fatal("loadFromDB() 未恢复 ModelStates")
	}

	state := acc.ModelStates["gpt-5"]
	if state == nil {
		t.Fatalf("loadFromDB() 未恢复 gpt-5 状态, got=%v", acc.ModelStates)
	}

	if !state.Unavailable {
		t.Fatal("gpt-5 状态应为 unavailable=true")
	}

	if acc.Status != StatusReady {
		t.Fatalf("单模型冷却时账号不应整体进入 cooldown, got=%v", acc.Status)
	}
}

func TestRecomputeAggregatedAccountStateUsesRoutableModels(t *testing.T) {
	now := time.Now().UTC()
	acc := &Account{
		AccessToken: "token-1",
		Status:      StatusReady,
		ModelStates: map[string]*ModelState{
			"gpt-5": {
				Status:         ModelStatusCooldown,
				Unavailable:    true,
				NextRetryAfter: now.Add(10 * time.Minute),
				UpdatedAt:      now,
			},
		},
	}

	acc.RecomputeAggregatedAccountState(now)

	if acc.Status != StatusReady {
		t.Fatalf("仅一个已知模型冷却时账号不应整体进入 cooldown, got=%v", acc.Status)
	}
}
