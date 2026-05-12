package admin

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/codex2api/database"
	"github.com/gin-gonic/gin"
)

const (
	maxAccountGroups            = 64
	maxAccountGroupNameRuneSize = 80
)

type accountGroupResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	SortOrder   int64  `json:"sort_order"`
	MemberCount int64  `json:"member_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type accountGroupsResponse struct {
	Groups []accountGroupResponse `json:"groups"`
}

func toAccountGroupResponse(g database.AccountGroup) accountGroupResponse {
	return accountGroupResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Color:       g.Color,
		SortOrder:   g.SortOrder,
		MemberCount: g.MemberCount,
		CreatedAt:   g.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   g.UpdatedAt.Format(time.RFC3339),
	}
}

// ListAccountGroups GET /api/admin/account-groups
func (h *Handler) ListAccountGroups(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	groups, err := h.db.ListAccountGroups(ctx)
	if err != nil {
		writeInternalError(c, err)
		return
	}
	out := make([]accountGroupResponse, 0, len(groups))
	for _, g := range groups {
		out = append(out, toAccountGroupResponse(g))
	}
	c.JSON(http.StatusOK, accountGroupsResponse{Groups: out})
}

type createAccountGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	SortOrder   *int64 `json:"sort_order"`
}

// CreateAccountGroup POST /api/admin/account-groups
func (h *Handler) CreateAccountGroup(c *gin.Context) {
	var req createAccountGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	name, err := sanitizeAccountGroupName(req.Name)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	description := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(description) > 240 {
		writeError(c, http.StatusBadRequest, "描述长度不能超过 240 字符")
		return
	}
	color := strings.TrimSpace(req.Color)
	if utf8.RuneCountInString(color) > 20 {
		writeError(c, http.StatusBadRequest, "颜色长度不能超过 20 字符")
		return
	}
	var sortOrder int64
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	groups, err := h.db.ListAccountGroups(ctx)
	if err != nil {
		writeInternalError(c, err)
		return
	}
	if len(groups) >= maxAccountGroups {
		writeError(c, http.StatusBadRequest, "分组数量已达上限")
		return
	}

	id, err := h.db.CreateAccountGroup(ctx, name, description, color, sortOrder)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateAccountGroupName) {
			writeError(c, http.StatusConflict, err.Error())
			return
		}
		writeInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "分组已创建"})
}

type updateAccountGroupReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
	SortOrder   *int64  `json:"sort_order"`
}

// UpdateAccountGroup PATCH /api/admin/account-groups/:id
func (h *Handler) UpdateAccountGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的分组 ID")
		return
	}
	var req updateAccountGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.Name != nil {
		clean, err := sanitizeAccountGroupName(*req.Name)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		req.Name = &clean
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if utf8.RuneCountInString(desc) > 240 {
			writeError(c, http.StatusBadRequest, "描述长度不能超过 240 字符")
			return
		}
		req.Description = &desc
	}
	if req.Color != nil {
		color := strings.TrimSpace(*req.Color)
		if utf8.RuneCountInString(color) > 20 {
			writeError(c, http.StatusBadRequest, "颜色长度不能超过 20 字符")
			return
		}
		req.Color = &color
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := h.db.UpdateAccountGroup(ctx, id, req.Name, req.Description, req.Color, req.SortOrder); err != nil {
		if err == sql.ErrNoRows {
			writeError(c, http.StatusNotFound, "分组不存在")
			return
		}
		if errors.Is(err, database.ErrDuplicateAccountGroupName) {
			writeError(c, http.StatusConflict, err.Error())
			return
		}
		writeInternalError(c, err)
		return
	}
	writeMessage(c, http.StatusOK, "分组已更新")
}

// DeleteAccountGroup DELETE /api/admin/account-groups/:id?force=true
func (h *Handler) DeleteAccountGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "无效的分组 ID")
		return
	}
	force := strings.EqualFold(c.Query("force"), "true")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := h.db.DeleteAccountGroup(ctx, id, force); err != nil {
		if err == sql.ErrNoRows {
			writeError(c, http.StatusNotFound, "分组不存在")
			return
		}
		if errors.Is(err, database.ErrAccountGroupNotEmpty) {
			writeError(c, http.StatusConflict, err.Error())
			return
		}
		writeInternalError(c, err)
		return
	}
	// 同步内存：清掉所有账号对该分组的引用
	if h.store != nil {
		for _, acc := range h.store.Accounts() {
			acc.GroupIDs = removeInt64(acc.GroupIDs, id)
		}
	}
	writeMessage(c, http.StatusOK, "分组已删除")
}

// sanitizeAccountGroupName trims and validates a group name. Returns the canonical form.
func sanitizeAccountGroupName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", errAccountGroupNameRequired
	}
	if utf8.RuneCountInString(name) > maxAccountGroupNameRuneSize {
		return "", errAccountGroupNameTooLong
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return "", errAccountGroupNameInvalid
		}
	}
	return name, nil
}

var (
	errAccountGroupNameRequired = errors.New("分组名称不能为空")
	errAccountGroupNameTooLong  = errors.New("分组名称长度超过 80 字符")
	errAccountGroupNameInvalid  = errors.New("分组名称包含非法控制字符")
)

func removeInt64(slice []int64, target int64) []int64 {
	out := slice[:0]
	for _, v := range slice {
		if v != target {
			out = append(out, v)
		}
	}
	return out
}

// dedupeInt64 removes duplicate values while preserving the order of first appearance.
// Negative or zero IDs are dropped.
func dedupeInt64(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
