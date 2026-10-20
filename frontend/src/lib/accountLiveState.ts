import type { AccountLiveStateResponse } from '../types'

// Account rows carry a large optional usage/status projection from the list
// API. The live poll refreshes the same fields in place so the card's bars,
// status badge and countdown react to per-turn WS usage observations without a
// list reload. Missing keys mean "older backend: keep the list value"; an
// explicit null means "the window is no longer valid: clear it".
export type AccountLiveMergeable = {
  id: number
  active_requests?: number
  occupied_requests?: number
  session_slot_buffer_enabled?: boolean
  dynamic_concurrency_limit?: number
  base_concurrency_effective?: number
  status?: string
  using_credits?: boolean
  plan_type?: string
  credits_valid?: boolean
  credits_balance?: string | null
  credits_has_credits?: boolean | null
  credits_unlimited?: boolean | null
  credits_overage_limit_reached?: boolean | null
  credits_spend_control_reached?: boolean | null
  credits_rate_limit_reached_type?: string | null
  usage_percent_5h?: number | null
  reset_5h_at?: string | null
  codex_5h_usage_updated_at?: string | null
  usage_percent_7d?: number | null
  reset_7d_at?: string | null
  codex_usage_updated_at?: string | null
  cooldown_until?: string | null
  cooldown_reason?: string
  quota_guard_pending?: boolean
  codex_usage_frame_at?: string | null
  codex_usage_frame_kind?: string
  openai_responses_api?: boolean
  grok_api?: boolean
  antigravity_api?: boolean
  claude_api?: boolean
  usage_5h_detail?: unknown
  usage_7d_detail?: unknown
  billed_5h?: number
  billed_7d?: number
}

// Keep only the visible page's latest observations. Partial responses retain
// earlier keys; explicit null is retained so a stale list cannot revive them.
export function mergeAccountLiveResponse(
  current: AccountLiveStateResponse | null,
  incoming: AccountLiveStateResponse,
): AccountLiveStateResponse {
  if (!current) return incoming
  let changed = current.session_slot_buffer_enabled !== incoming.session_slot_buffer_enabled
  const accounts = { ...current.accounts }
  for (const [id, item] of Object.entries(incoming.accounts)) {
    const previous = current.accounts[id]
    const fields = Object.fromEntries(Object.entries(item).filter(([, value]) => value !== undefined))
    if (previous && Object.entries(fields).every(([key, value]) => previous[key as keyof typeof previous] === value)) continue
    accounts[id] = { ...previous, ...fields } as typeof item
    changed = true
  }
  return changed ? { ...incoming, accounts } : current
}

export function getAccountUsageUpdatedAt(account: AccountLiveMergeable): string | undefined {
  return [account.codex_5h_usage_updated_at, account.codex_usage_updated_at]
    .filter((value): value is string => Boolean(value) && Number.isFinite(Date.parse(value!)))
    .sort((a, b) => Date.parse(b) - Date.parse(a))[0]
}

export function mergeAccountLiveState<T extends AccountLiveMergeable>(
  current: T[],
  response: AccountLiveStateResponse,
): T[] {
  let changed = false
  const next = current.map((account) => {
    const live = response.accounts[String(account.id)]
    if (!live) return account

    const activeRequests = live.active_requests ?? 0
    const occupiedRequests = live.occupied_requests ?? activeRequests
    const slotBufferEnabled = response.session_slot_buffer_enabled === true
    // Concurrency caps are optional so an older backend keeps the list values.
    const dynamicLimit = typeof live.dynamic_concurrency_limit === 'number'
      ? live.dynamic_concurrency_limit
      : account.dynamic_concurrency_limit
    const baseConcurrency = typeof live.base_concurrency_effective === 'number'
      ? live.base_concurrency_effective
      : account.base_concurrency_effective
    // Usage/status fields follow the same contract: undefined keeps the row,
    // null clears it. The list API omits some fields for Antigravity rows.
    const status = live.status === undefined ? account.status : live.status
    const usingCredits = live.using_credits === undefined
      ? account.using_credits
      : live.using_credits
    const usage5h = live.usage_percent_5h === undefined
      ? account.usage_percent_5h
      : live.usage_percent_5h
    const reset5h = live.reset_5h_at === undefined
      ? account.reset_5h_at
      : (live.reset_5h_at ?? undefined)
    const updated5h = live.codex_5h_usage_updated_at === undefined
      ? account.codex_5h_usage_updated_at
      : (live.codex_5h_usage_updated_at ?? undefined)
    const usage7d = live.usage_percent_7d === undefined
      ? account.usage_percent_7d
      : live.usage_percent_7d
    const reset7d = live.reset_7d_at === undefined
      ? account.reset_7d_at
      : (live.reset_7d_at ?? undefined)
    const updated7d = live.codex_usage_updated_at === undefined
      ? account.codex_usage_updated_at
      : (live.codex_usage_updated_at ?? undefined)
    const cooldownUntil = live.cooldown_until === undefined
      ? account.cooldown_until
      : (live.cooldown_until ?? undefined)
    const quotaGuardPending = live.quota_guard_pending === undefined
      ? account.quota_guard_pending
      : live.quota_guard_pending
    const frameAt = live.codex_usage_frame_at === undefined
      ? account.codex_usage_frame_at
      : (live.codex_usage_frame_at ?? undefined)
    // Gateway-only accounts have local rolling stats, not official quota
    // windows. A cleared OAuth window must not reappear through page-stats.
    const officialWindows = !account.openai_responses_api && !account.grok_api &&
      !account.antigravity_api && !account.claude_api
    const clear5h = officialWindows && live.usage_percent_5h === null && live.reset_5h_at === null
    const clear7d = officialWindows && live.usage_percent_7d === null && live.reset_7d_at === null
    // Existing plan/credit badges use the list API's field names and null
    // semantics. A hidden balance must clear an older visible value.
    const details = {
      plan_type: live.plan_type === undefined ? account.plan_type : live.plan_type,
      credits_valid: live.credits_valid === undefined ? account.credits_valid : live.credits_valid,
      credits_balance: live.credits_balance === undefined ? account.credits_balance : live.credits_balance,
      credits_has_credits: live.credits_has_credits === undefined ? account.credits_has_credits : live.credits_has_credits,
      credits_unlimited: live.credits_unlimited === undefined ? account.credits_unlimited : live.credits_unlimited,
      credits_overage_limit_reached: live.credits_overage_limit_reached === undefined ? account.credits_overage_limit_reached : live.credits_overage_limit_reached,
      credits_spend_control_reached: live.credits_spend_control_reached === undefined ? account.credits_spend_control_reached : live.credits_spend_control_reached,
      credits_rate_limit_reached_type: live.credits_rate_limit_reached_type === undefined ? account.credits_rate_limit_reached_type : live.credits_rate_limit_reached_type,
      cooldown_until: cooldownUntil,
      cooldown_reason: live.cooldown_reason === undefined ? account.cooldown_reason : live.cooldown_reason,
      quota_guard_pending: quotaGuardPending,
      codex_usage_frame_at: frameAt,
      codex_usage_frame_kind: live.codex_usage_frame_kind === undefined ? account.codex_usage_frame_kind : live.codex_usage_frame_kind,
      usage_5h_detail: clear5h ? undefined : account.usage_5h_detail,
      usage_7d_detail: clear7d ? undefined : account.usage_7d_detail,
      billed_5h: clear5h ? undefined : account.billed_5h,
      billed_7d: clear7d ? undefined : account.billed_7d,
    }
    const detailsUnchanged = (Object.keys(details) as (keyof typeof details)[])
      .every((key) => account[key] === details[key])

    if (
      detailsUnchanged &&
      (account.active_requests ?? 0) === activeRequests &&
      (account.occupied_requests ?? account.active_requests ?? 0) === occupiedRequests &&
      (account.session_slot_buffer_enabled ?? false) === slotBufferEnabled &&
      account.dynamic_concurrency_limit === dynamicLimit &&
      account.base_concurrency_effective === baseConcurrency &&
      account.status === status &&
      account.using_credits === usingCredits &&
      account.usage_percent_5h === usage5h &&
      account.reset_5h_at === reset5h &&
      account.codex_5h_usage_updated_at === updated5h &&
      account.usage_percent_7d === usage7d &&
      account.reset_7d_at === reset7d &&
      account.codex_usage_updated_at === updated7d
    ) return account

    changed = true
    return {
      ...account,
      ...details,
      active_requests: activeRequests,
      occupied_requests: occupiedRequests,
      session_slot_buffer_enabled: slotBufferEnabled,
      dynamic_concurrency_limit: dynamicLimit,
      base_concurrency_effective: baseConcurrency,
      status,
      using_credits: usingCredits,
      usage_percent_5h: usage5h,
      reset_5h_at: reset5h,
      codex_5h_usage_updated_at: updated5h,
      usage_percent_7d: usage7d,
      reset_7d_at: reset7d,
      codex_usage_updated_at: updated7d,
    }
  })
  return changed ? next : current
}
