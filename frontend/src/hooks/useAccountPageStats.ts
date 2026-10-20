import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'
import type { AccountPageStatsItem } from '../types'
import { ACCOUNT_PAGE_STATS_MIN_REFRESH_MS } from '../lib/accountListPreferences'

export function useAccountPageStats(idsKey: string, reloadToken: number, enabled = true) {
  const [snapshot, setSnapshot] = useState<{
    idsKey: string
    stats: Record<string, AccountPageStatsItem>
    updatedAt: string
  } | null>(null)

  useEffect(() => {
    if (!enabled || !idsKey) return undefined
    let stopped = false
    let timer: number | undefined
    let active: { controller: AbortController; timeout: number } | undefined
    const clearTimer = () => {
      window.clearTimeout(timer)
      timer = undefined
    }
    const finish = (request: NonNullable<typeof active>) => {
      if (active !== request) return
      window.clearTimeout(request.timeout)
      active = undefined
      if (!stopped && !document.hidden) timer = window.setTimeout(poll, ACCOUNT_PAGE_STATS_MIN_REFRESH_MS)
    }
    const cancel = () => {
      clearTimer()
      if (!active) return
      const request = active
      active = undefined
      window.clearTimeout(request.timeout)
      request.controller.abort()
    }
    const poll = () => {
      if (stopped || document.hidden || active) return
      clearTimer()
      const controller = new AbortController()
      const request = { controller, timeout: 0 }
      active = request
      request.timeout = window.setTimeout(() => {
        controller.abort()
        finish(request)
      }, ACCOUNT_PAGE_STATS_MIN_REFRESH_MS)
      void api.getAccountPageStats(idsKey.split(',').map(Number), controller.signal)
        .then((response) => {
          if (!stopped && active === request && !controller.signal.aborted) {
            setSnapshot({ idsKey, stats: response.stats ?? {}, updatedAt: new Date().toISOString() })
          }
        })
        .catch((error: unknown) => {
          if (!controller.signal.aborted) console.warn('account page stats load failed:', error)
        })
        .finally(() => finish(request))
    }
    const recover = () => {
      if (document.hidden) cancel()
      else if (timer === undefined) poll()
    }
    document.addEventListener('visibilitychange', recover)
    window.addEventListener('focus', recover)
    poll()
    return () => {
      stopped = true
      cancel()
      document.removeEventListener('visibilitychange', recover)
      window.removeEventListener('focus', recover)
    }
  }, [idsKey, reloadToken, enabled])

  const patchOfficialUsage = useCallback((patch: { accountId: number; officialUsd: number | null }) => {
    setSnapshot((current) => {
      if (!current || current.idsKey !== idsKey) return current
      const key = String(patch.accountId)
      return {
        ...current,
        stats: {
          ...current.stats,
          [key]: {
            ...current.stats[key],
            official_usd: patch.officialUsd ?? undefined,
            official_usd_7d: patch.officialUsd ?? undefined,
            official_usage_synced: true,
          },
        },
      }
    })
  }, [idsKey])
  return {
    snapshot: enabled && snapshot?.idsKey === idsKey ? snapshot : null,
    patchOfficialUsage,
  }
}
