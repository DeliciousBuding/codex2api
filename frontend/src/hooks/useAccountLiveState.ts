import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { api } from '../api'
import type { AccountLiveStateResponse } from '../types'
import { mergeAccountLiveResponse, mergeAccountLiveState, type AccountLiveMergeable } from '../lib/accountLiveState'

export { mergeAccountLiveState }

export function useAccountLiveState(
  ids: number[],
  apply: (response: AccountLiveStateResponse) => void,
  enabled = true,
) {
  const idsKey = useMemo(() => ids.join(','), [ids])

  useEffect(() => {
    if (!enabled || !idsKey) return undefined
    let stopped = false
    let timer: number | undefined
    let active: { controller: AbortController; timeout: number } | undefined

    const clearTimer = () => {
      window.clearTimeout(timer)
      timer = undefined
    }
    const schedule = () => {
      clearTimer()
      if (!stopped && !document.hidden) timer = window.setTimeout(poll, 1000)
    }
    const finish = (request: NonNullable<typeof active>) => {
      if (active !== request) return
      window.clearTimeout(request.timeout)
      active = undefined
      schedule()
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
      // Release the poll even if a transport never settles after abort. Late
      // responses are ignored, so only one non-aborted request owns the page.
      request.timeout = window.setTimeout(() => {
        controller.abort()
        finish(request)
      }, 5000)
      void api.getAccountLiveState(idsKey.split(',').map(Number), controller.signal)
        .then((response) => {
          if (!stopped && active === request && !controller.signal.aborted) apply(response)
        })
        .catch(() => undefined)
        .finally(() => finish(request))
    }
    const recover = () => {
      if (document.hidden) cancel()
      else poll()
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
  }, [apply, enabled, idsKey])
}

// Runtime fields remain independent of list/page-stats requests. Applying the
// overlay last prevents a delayed list response from regressing a newer poll.
export function useLiveAccounts<T extends AccountLiveMergeable>(rows: T[], enabled = true): T[] {
  const ids = useMemo(() => rows.map(({ id }) => id), [rows])
  const idsKey = ids.join(',')
  const scope = useMemo(() => ({ idsKey, enabled }), [idsKey, enabled])
  const [snapshot, setSnapshot] = useState<{ scope: typeof scope; response: AccountLiveStateResponse } | null>(null)
  const apply = useCallback((response: AccountLiveStateResponse) => {
    setSnapshot((current) => {
      const previous = current?.scope === scope ? current.response : null
      const merged = mergeAccountLiveResponse(previous, response)
      return merged === previous ? current : { scope, response: merged }
    })
  }, [scope])
  useAccountLiveState(ids, apply, enabled)

  type CachedRow = {
    base: T
    live: AccountLiveStateResponse['accounts'][string] | undefined
    slotBuffer: boolean
    result: T
  }
  const cache = useRef(new Map<number, CachedRow>())
  useEffect(() => {
    setSnapshot((current) => current?.scope === scope ? current : null)
  }, [scope])
  return useMemo(() => {
    if (!enabled || snapshot?.scope !== scope) {
      cache.current.clear()
      return rows
    }
    const response = snapshot.response
    const nextCache = new Map<number, CachedRow>()
    const merged = rows.map((base) => {
      const live = response.accounts[String(base.id)]
      const slotBuffer = response.session_slot_buffer_enabled
      const previous = cache.current.get(base.id)
      const result = previous?.base === base && previous.live === live && previous.slotBuffer === slotBuffer
        ? previous.result
        : mergeAccountLiveState([base], response)[0]
      nextCache.set(base.id, { base, live, slotBuffer, result })
      return result
    })
    cache.current = nextCache
    return merged.every((row, index) => row === rows[index]) ? rows : merged
  }, [rows, enabled, scope, snapshot])
}
