import type { AccountPageStatsItem, AccountRow } from '../types'

// Page stats own gateway request/token/cost totals. They never overwrite the
// quota percentages or runtime status, which are supplied by the live overlay.
export function mergeAccountPageStats(
  accounts: AccountRow[],
  statsByID: Record<string, AccountPageStatsItem>,
  updatedAt?: string,
): AccountRow[] {
  let changed = false
  const next = accounts.map((account) => {
    const stats = statsByID[String(account.id)]
    if (!stats) return account
    const officialUsd = stats.official_usd ?? stats.official_usd_7d
    const fields = {
      billed_5h: stats.billed_5h ?? account.billed_5h,
      billed_7d: stats.billed_7d ?? account.billed_7d,
      usage_5h_detail: stats.usage_5h_detail ?? account.usage_5h_detail,
      usage_7d_detail: stats.usage_7d_detail ?? account.usage_7d_detail,
      usage_today_detail: stats.usage_today_detail ?? account.usage_today_detail,
      official_usd: officialUsd ?? (stats.official_usage_synced ? undefined : account.official_usd),
      official_usd_7d: officialUsd ?? (stats.official_usage_synced ? undefined : account.official_usd_7d),
      official_usage_synced: stats.official_usage_synced ?? account.official_usage_synced,
      gateway_usage_updated_at: updatedAt,
    }
    if (Object.entries(fields).every(([key, value]) => account[key as keyof AccountRow] === value)) return account
    changed = true
    return { ...account, ...fields }
  })
  return changed ? next : accounts
}
