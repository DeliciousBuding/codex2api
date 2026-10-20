import assert from 'node:assert/strict'
import test from 'node:test'
import { mergeAccountPageStats } from './accountPageStats.ts'

test('gateway snapshots update detail totals without overwriting official quota/status', () => {
  const row = { id: 1, status: 'quota_paused', usage_percent_5h: 95, billed_5h: 1, usage_5h_detail: { requests: 2 } }
  const stats = { "1": { billed_5h: 2, usage_5h_detail: { requests: 3 }, official_usd: 12, official_usage_synced: true } }
  const merged = mergeAccountPageStats([row], stats, '2026-10-20T05:00:00Z')[0]
  assert.equal(merged.status, 'quota_paused')
  assert.equal(merged.usage_percent_5h, 95)
  assert.equal(merged.billed_5h, 2)
  assert.equal(merged.usage_5h_detail.requests, 3)
  assert.equal(merged.official_usd, 12)
  assert.equal(merged.gateway_usage_updated_at, '2026-10-20T05:00:00Z')
})

test('unchanged stats retain row identity and unknown official settlement never becomes zero', () => {
  const detail = { requests: 2 }
  const row = { id: 1, billed_5h: 1, usage_5h_detail: detail }
  const rows = [row]
  assert.equal(mergeAccountPageStats(rows, { "1": { billed_5h: 1, usage_5h_detail: detail } }), rows)
  const stale = { ...row, official_usd: 12, official_usd_7d: 12 }
  const next = mergeAccountPageStats([stale], { "1": { official_usage_synced: true } })[0]
  assert.equal(next.official_usd, undefined)
  assert.equal(next.official_usd_7d, undefined)
})
