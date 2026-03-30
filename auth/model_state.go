package auth

import (
	"strings"
	"time"
)

type ModelStatus int

const (
	ModelStatusActive ModelStatus = iota
	ModelStatusCooldown
	ModelStatusUnsupported
)

type ModelState struct {
	Status         ModelStatus `json:"status"`
	Unavailable    bool        `json:"unavailable"`
	NextRetryAfter time.Time   `json:"next_retry_after,omitempty"`
	LastError      string      `json:"last_error,omitempty"`
	StrikeCount    int         `json:"strike_count"`
	BackoffLevel   int         `json:"backoff_level"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func canonicalModelKey(model string) string {
	return strings.TrimSpace(model)
}

func (m *ModelState) IsBlocked(now time.Time) bool {
	if m == nil || !m.Unavailable {
		return false
	}
	if m.NextRetryAfter.IsZero() {
		return false
	}
	return now.Before(m.NextRetryAfter)
}

func (a *Account) RecomputeAggregatedAccountState(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.recomputeAggregatedAccountStateLocked(now)
}

func (a *Account) recomputeAggregatedAccountStateLocked(now time.Time) {
	if a.Status == StatusError || a.healthTierLocked() == HealthTierBanned {
		return
	}

	allBlocked := true
	var earliest time.Time

	for _, model := range routableModelKeys() {
		key := canonicalModelKey(model)
		state := a.ModelStates[key]
		if state == nil || !state.IsBlocked(now) {
			allBlocked = false
			continue
		}
		if earliest.IsZero() || state.NextRetryAfter.Before(earliest) {
			earliest = state.NextRetryAfter
		}
	}

	if allBlocked && !earliest.IsZero() {
		a.Status = StatusCooldown
		a.CooldownUtil = earliest
		a.CooldownReason = "all_models_rate_limited"
		return
	}

	if a.Status == StatusCooldown && a.CooldownReason == "all_models_rate_limited" {
		a.Status = StatusReady
		a.CooldownUtil = time.Time{}
		a.CooldownReason = ""
	}
}
