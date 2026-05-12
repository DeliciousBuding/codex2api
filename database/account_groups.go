package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AccountGroup is a named collection of accounts. Membership is many-to-many.
type AccountGroup struct {
	ID          int64
	Name        string
	Description string
	Color       string
	SortOrder   int64
	MemberCount int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListAccountGroups returns all groups ordered by sort_order then name.
// MemberCount is computed via a left join against account_group_members.
func (db *DB) ListAccountGroups(ctx context.Context) ([]AccountGroup, error) {
	query := `
		SELECT g.id, g.name, g.description, g.color, g.sort_order,
			COALESCE(COUNT(m.account_id), 0) AS member_count,
			g.created_at, g.updated_at
		FROM account_groups g
		LEFT JOIN account_group_members m ON m.group_id = g.id
		GROUP BY g.id, g.name, g.description, g.color, g.sort_order, g.created_at, g.updated_at
		ORDER BY g.sort_order, g.name
	`
	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询账号分组失败: %w", err)
	}
	defer rows.Close()
	var out []AccountGroup
	for rows.Next() {
		var g AccountGroup
		var createdAtRaw, updatedAtRaw interface{}
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Color, &g.SortOrder, &g.MemberCount, &createdAtRaw, &updatedAtRaw); err != nil {
			return nil, fmt.Errorf("扫描分组行失败: %w", err)
		}
		if g.CreatedAt, err = parseDBTimeValue(createdAtRaw); err != nil {
			return nil, fmt.Errorf("解析 created_at 失败: %w", err)
		}
		if g.UpdatedAt, err = parseDBTimeValue(updatedAtRaw); err != nil {
			return nil, fmt.Errorf("解析 updated_at 失败: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// CreateAccountGroup inserts a new group. Returns the new id or a wrapped error.
// On unique constraint violation returns ErrDuplicateAccountGroupName.
func (db *DB) CreateAccountGroup(ctx context.Context, name, description, color string, sortOrder int64) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("分组名称不能为空")
	}
	if db.isSQLite() {
		res, err := db.conn.ExecContext(ctx,
			`INSERT INTO account_groups (name, description, color, sort_order) VALUES (?, ?, ?, ?)`,
			name, description, color, sortOrder)
		if err != nil {
			if isUniqueViolation(err) {
				return 0, ErrDuplicateAccountGroupName
			}
			return 0, err
		}
		return res.LastInsertId()
	}
	var id int64
	err := db.conn.QueryRowContext(ctx,
		`INSERT INTO account_groups (name, description, color, sort_order) VALUES ($1, $2, $3, $4) RETURNING id`,
		name, description, color, sortOrder).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateAccountGroupName
		}
		return 0, err
	}
	return id, nil
}

// UpdateAccountGroup partially updates a group's editable fields.
// Nil pointers leave the field unchanged.
func (db *DB) UpdateAccountGroup(ctx context.Context, id int64, name, description, color *string, sortOrder *int64) error {
	sets := make([]string, 0, 5)
	args := make([]interface{}, 0, 5)
	idx := 1
	placeholder := func() string {
		if db.isSQLite() {
			return "?"
		}
		s := fmt.Sprintf("$%d", idx)
		idx++
		return s
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return fmt.Errorf("分组名称不能为空")
		}
		sets = append(sets, "name = "+placeholder())
		args = append(args, trimmed)
	}
	if description != nil {
		sets = append(sets, "description = "+placeholder())
		args = append(args, *description)
	}
	if color != nil {
		sets = append(sets, "color = "+placeholder())
		args = append(args, *color)
	}
	if sortOrder != nil {
		sets = append(sets, "sort_order = "+placeholder())
		args = append(args, *sortOrder)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	whereParam := placeholder()
	args = append(args, id)
	query := "UPDATE account_groups SET " + strings.Join(sets, ", ") + " WHERE id = " + whereParam
	res, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateAccountGroupName
		}
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteAccountGroup removes a group and its membership rows.
// Returns sql.ErrNoRows if the group does not exist.
// If members > 0 and force is false, returns ErrAccountGroupNotEmpty.
func (db *DB) DeleteAccountGroup(ctx context.Context, id int64, force bool) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var count int64
	countQuery := "SELECT COUNT(*) FROM account_group_members WHERE group_id = $1"
	if db.isSQLite() {
		countQuery = "SELECT COUNT(*) FROM account_group_members WHERE group_id = ?"
	}
	if err := tx.QueryRowContext(ctx, countQuery, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 && !force {
		return ErrAccountGroupNotEmpty
	}

	delMembersQ := "DELETE FROM account_group_members WHERE group_id = $1"
	delGroupQ := "DELETE FROM account_groups WHERE id = $1"
	if db.isSQLite() {
		delMembersQ = "DELETE FROM account_group_members WHERE group_id = ?"
		delGroupQ = "DELETE FROM account_groups WHERE id = ?"
	}
	if _, err := tx.ExecContext(ctx, delMembersQ, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, delGroupQ, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

// SetAccountGroups replaces the group membership for one account in a single tx.
// Passing an empty slice clears all memberships.
func (db *DB) SetAccountGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	delQ := "DELETE FROM account_group_members WHERE account_id = $1"
	if db.isSQLite() {
		delQ = "DELETE FROM account_group_members WHERE account_id = ?"
	}
	if _, err := tx.ExecContext(ctx, delQ, accountID); err != nil {
		return err
	}
	if len(groupIDs) > 0 {
		seen := make(map[int64]struct{}, len(groupIDs))
		insQ := "INSERT INTO account_group_members (account_id, group_id) VALUES ($1, $2)"
		if db.isSQLite() {
			insQ = "INSERT INTO account_group_members (account_id, group_id) VALUES (?, ?)"
		}
		for _, gid := range groupIDs {
			if _, dup := seen[gid]; dup {
				continue
			}
			seen[gid] = struct{}{}
			if _, err := tx.ExecContext(ctx, insQ, accountID, gid); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// GetAccountGroupIDs returns the group IDs an account belongs to.
func (db *DB) GetAccountGroupIDs(ctx context.Context, accountID int64) ([]int64, error) {
	query := "SELECT group_id FROM account_group_members WHERE account_id = $1 ORDER BY group_id"
	if db.isSQLite() {
		query = "SELECT group_id FROM account_group_members WHERE account_id = ? ORDER BY group_id"
	}
	rows, err := db.conn.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var gid int64
		if err := rows.Scan(&gid); err != nil {
			return nil, err
		}
		out = append(out, gid)
	}
	return out, rows.Err()
}

// ListAccountGroupMemberships returns a map of accountID -> []groupID.
// Used at startup to hydrate the in-memory Store.
func (db *DB) ListAccountGroupMemberships(ctx context.Context) (map[int64][]int64, error) {
	rows, err := db.conn.QueryContext(ctx, `SELECT account_id, group_id FROM account_group_members ORDER BY account_id, group_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64][]int64)
	for rows.Next() {
		var aid, gid int64
		if err := rows.Scan(&aid, &gid); err != nil {
			return nil, err
		}
		out[aid] = append(out[aid], gid)
	}
	return out, rows.Err()
}

// VerifyAccountGroupIDs returns the subset of input IDs that do NOT exist.
func (db *DB) VerifyAccountGroupIDs(ctx context.Context, ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	placeholders := make([]string, len(unique))
	args := make([]interface{}, len(unique))
	for i, id := range unique {
		if db.isSQLite() {
			placeholders[i] = "?"
		} else {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		args[i] = id
	}
	query := fmt.Sprintf("SELECT id FROM account_groups WHERE id IN (%s)", strings.Join(placeholders, ", "))
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	existing := make(map[int64]struct{}, len(unique))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		existing[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	missing := make([]int64, 0)
	for _, id := range unique {
		if _, ok := existing[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

// ErrDuplicateAccountGroupName is returned when CreateAccountGroup or
// UpdateAccountGroup violates the unique constraint on name.
var ErrDuplicateAccountGroupName = fmt.Errorf("分组名称已存在")

// ErrAccountGroupNotEmpty is returned when DeleteAccountGroup is called on
// a group with members and force=false.
var ErrAccountGroupNotEmpty = fmt.Errorf("分组下仍有账号，需先解除关联或使用 force=true")

// isUniqueViolation returns true if the error looks like a unique constraint
// failure for either Postgres (SQLSTATE 23505) or SQLite (constraint failed).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate key") || strings.Contains(msg, "23505")
}
