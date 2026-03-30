package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteAccountsTableIncludesModelStatesColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "codex2api.db")

	db, err := New("sqlite", dbPath)
	if err != nil {
		t.Fatalf("New(sqlite) 返回错误: %v", err)
	}
	defer db.Close()

	columns, err := db.sqliteTableColumns(context.Background(), "accounts")
	if err != nil {
		t.Fatalf("sqliteTableColumns(accounts) 返回错误: %v", err)
	}

	if _, ok := columns["model_states"]; !ok {
		t.Fatalf("accounts 表缺少 model_states 列, got=%v", columns)
	}
}
