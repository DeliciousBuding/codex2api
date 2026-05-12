import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Globe, Plus, Trash2, Play, MapPin, Loader2, Zap, ChevronLeft, ChevronRight, Eye, EyeOff, AlertTriangle } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { api, type ProxyRow, type ProxyTestResult } from '../api'
import ToastNotice from '../components/ToastNotice'
import { useToast } from '../hooks/useToast'
import { getErrorMessage } from '../utils/error'

const PAGE_SIZE = 10
const SLOW_PROXY_MS = 1500
const TEST_ALL_CONCURRENCY = 4
type ProxyFilter = 'all' | 'enabled' | 'disabled' | 'untested' | 'slow'

function latencyColor(ms: number): string {
  if (ms <= 0) return 'text-muted-foreground'
  if (ms < 500) return 'text-emerald-600 dark:text-emerald-400'
  if (ms < 1500) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

function latencyBg(ms: number): string {
  if (ms <= 0) return ''
  if (ms < 500) return 'bg-emerald-500/10'
  if (ms < 1500) return 'bg-amber-500/10'
  return 'bg-red-500/10'
}

function maskUrl(url: string): string {
  try {
    const u = new URL(url)
    const host = u.hostname
    const masked = host.length > 6 ? host.slice(0, 3) + '***' + host.slice(-3) : '***'
    return `${u.protocol}//${u.username ? '***:***@' : ''}${masked}${u.port ? ':' + u.port : ''}`
  } catch {
    return url.slice(0, 10) + '******'
  }
}

function validateProxyInput(url: string): boolean {
  try {
    const parsed = new URL(url)
    return Boolean(parsed.hostname) && ['http:', 'https:', 'socks5:', 'socks5h:'].includes(parsed.protocol)
  } catch {
    return false
  }
}

export default function Proxies() {
  const { t, i18n } = useTranslation()
  const { toast, showToast } = useToast()
  const [proxies, setProxies] = useState<ProxyRow[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [poolEnabled, setPoolEnabled] = useState(false)
  const [showAdd, setShowAdd] = useState(false)
  const [addInput, setAddInput] = useState('')
  const [addLabel, setAddLabel] = useState('')
  const [addLoading, setAddLoading] = useState(false)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [testingIds, setTestingIds] = useState<Set<number>>(new Set())
  const [testAllLoading, setTestAllLoading] = useState(false)
  const [testAllDone, setTestAllDone] = useState(0)
  const [testAllFailed, setTestAllFailed] = useState(0)
  const [page, setPage] = useState(1)
  const [revealedIds, setRevealedIds] = useState<Set<number>>(new Set())
  const [filter, setFilter] = useState<ProxyFilter>('all')
  const [addError, setAddError] = useState('')

  const ipApiLang = i18n.language?.startsWith('zh') ? 'zh-CN' : 'en'

  const reload = useCallback(async () => {
    try {
      setLoadError('')
      const [proxyRes, settingsRes] = await Promise.all([api.listProxies(), api.getSettings()])
      setProxies(proxyRes.proxies)
      setPoolEnabled(settingsRes.proxy_pool_enabled)
    } catch (error) {
      const message = getErrorMessage(error)
      setLoadError(message)
      showToast(t('proxies.loadFailed', { error: message }), 'error')
    } finally {
      setLoading(false)
    }
  }, [showToast, t])

  useEffect(() => { reload() }, [reload])

  const proxyCounts = useMemo(() => ({
    total: proxies.length,
    enabled: proxies.filter(p => p.enabled).length,
    disabled: proxies.filter(p => !p.enabled).length,
    untested: proxies.filter(p => p.test_latency_ms <= 0).length,
    slow: proxies.filter(p => p.test_latency_ms >= SLOW_PROXY_MS).length,
  }), [proxies])

  const filteredProxies = useMemo(() => {
    switch (filter) {
      case 'enabled':
        return proxies.filter(p => p.enabled)
      case 'disabled':
        return proxies.filter(p => !p.enabled)
      case 'untested':
        return proxies.filter(p => p.test_latency_ms <= 0)
      case 'slow':
        return proxies.filter(p => p.test_latency_ms >= SLOW_PROXY_MS)
      default:
        return proxies
    }
  }, [filter, proxies])

  const totalPages = Math.max(1, Math.ceil(filteredProxies.length / PAGE_SIZE))
  const pagedProxies = filteredProxies.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  useEffect(() => {
    if (page > totalPages) setPage(totalPages)
  }, [page, totalPages])

  const handleTogglePool = async () => {
    const next = !poolEnabled
    setPoolEnabled(next)
    try {
      await api.updateSettings({ proxy_pool_enabled: next })
    } catch (error) {
      setPoolEnabled(!next)
      showToast(t('proxies.poolToggleFailed', { error: getErrorMessage(error) }), 'error')
    }
  }

  const handleAdd = async () => {
    const urls = addInput.split('\n').map(s => s.trim()).filter(Boolean)
    if (urls.length === 0) return
    const invalidURLs = urls.filter(url => !validateProxyInput(url))
    if (invalidURLs.length > 0) {
      setAddError(t('proxies.invalidProxyUrls', { count: invalidURLs.length }))
      return
    }
    setAddError('')
    setAddLoading(true)
    try {
      await api.addProxies({ urls, label: addLabel })
      setAddInput('')
      setAddLabel('')
      setShowAdd(false)
      await reload()
    } catch (error) {
      setAddError(error instanceof Error ? error.message : t('proxies.addFailed'))
    }
    setAddLoading(false)
  }

  const handleDelete = async (id: number) => {
    try {
      await api.deleteProxy(id)
      await reload()
    } catch (error) {
      showToast(t('proxies.deleteFailed', { error: getErrorMessage(error) }), 'error')
    }
  }

  const handleBatchDelete = async () => {
    if (selected.size === 0) return
    try {
      await api.batchDeleteProxies([...selected])
      setSelected(new Set())
      await reload()
    } catch (error) {
      showToast(t('proxies.batchDeleteFailed', { error: getErrorMessage(error) }), 'error')
    }
  }

  const handleToggle = async (p: ProxyRow) => {
    try {
      await api.updateProxy(p.id, { enabled: !p.enabled })
      await reload()
    } catch (error) {
      showToast(t('proxies.updateFailed', { error: getErrorMessage(error) }), 'error')
    }
  }

  const handleTest = async (p: ProxyRow) => {
    setTestingIds(prev => new Set(prev).add(p.id))
    try {
      const result = await api.testProxy(p.url, p.id, ipApiLang)
      if (result.success) {
        setProxies(prev => prev.map(px =>
          px.id === p.id
            ? { ...px, test_ip: result.ip || '', test_location: result.location || '', test_latency_ms: result.latency_ms || 0 }
            : px
        ))
      }
    } catch (error) {
      showToast(t('proxies.testFailed', { error: getErrorMessage(error) }), 'error')
    }
    setTestingIds(prev => {
      const next = new Set(prev)
      next.delete(p.id)
      return next
    })
  }

  const handleTestAll = async () => {
    setTestAllLoading(true)
    setTestAllDone(0)
    setTestAllFailed(0)
    let failedCount = 0
    let firstError = ''
    let nextIndex = 0
    const queue = [...proxies]
    const testOne = async (p: ProxyRow) => {
      setTestingIds(prev => new Set(prev).add(p.id))
      try {
        const result = await api.testProxy(p.url, p.id, ipApiLang)
        if (result.success) {
          setProxies(prev => prev.map(px =>
            px.id === p.id
              ? { ...px, test_ip: result.ip || '', test_location: result.location || '', test_latency_ms: result.latency_ms || 0 }
              : px
          ))
        }
      } catch (error) {
        failedCount += 1
        setTestAllFailed(failedCount)
        if (!firstError) firstError = getErrorMessage(error)
      } finally {
        setTestAllDone(prev => prev + 1)
        setTestingIds(prev => {
          const next = new Set(prev)
          next.delete(p.id)
          return next
        })
      }
    }

    const worker = async () => {
      for (;;) {
        const current = nextIndex
        nextIndex += 1
        const proxy = queue[current]
        if (!proxy) return
        await testOne(proxy)
      }
    }

    await Promise.all(Array.from({ length: Math.min(TEST_ALL_CONCURRENCY, queue.length) }, worker))
    if (failedCount > 0) {
      showToast(t('proxies.testAllFailed', { count: failedCount, error: firstError }), 'error')
    }
    setTestAllLoading(false)
  }

  const allSelected = pagedProxies.length > 0 && pagedProxies.every(p => selected.has(p.id))
  const toggleSelectAll = () => {
    if (allSelected) {
      setSelected(prev => {
        const next = new Set(prev)
        pagedProxies.forEach(p => next.delete(p.id))
        return next
      })
    } else {
      setSelected(prev => {
        const next = new Set(prev)
        pagedProxies.forEach(p => next.add(p.id))
        return next
      })
    }
  }

  const canEnable = proxyCounts.enabled > 0
  const filterOptions: Array<{ value: ProxyFilter; label: string; count: number }> = [
    { value: 'all', label: t('proxies.filterAll'), count: proxyCounts.total },
    { value: 'enabled', label: t('proxies.filterEnabled'), count: proxyCounts.enabled },
    { value: 'disabled', label: t('proxies.filterDisabled'), count: proxyCounts.disabled },
    { value: 'untested', label: t('proxies.filterUntested'), count: proxyCounts.untested },
    { value: 'slow', label: t('proxies.filterSlow'), count: proxyCounts.slow },
  ]

  return (
    <div className="space-y-6">
      <ToastNotice toast={toast} />
      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h2 className="text-2xl font-bold text-foreground flex items-center gap-2.5">
            <Globe className="size-6 text-primary" />
            {t('nav.proxies')}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('proxies.description')}
          </p>
        </div>
        <div className="flex flex-wrap items-center justify-end gap-2">
          {/* Pool Toggle Switch */}
          <div className="flex items-center gap-3" title={!canEnable && !poolEnabled ? t('proxies.addFirstProxy') : undefined}>
            <span className={`text-sm font-medium ${poolEnabled ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground'}`}>
              {poolEnabled ? t('proxies.poolEnabled') : t('proxies.poolDisabled')}
            </span>
            <button
              role="switch"
              aria-checked={poolEnabled}
              disabled={!canEnable && !poolEnabled}
              onClick={handleTogglePool}
              className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 disabled:cursor-not-allowed disabled:opacity-40 ${
                poolEnabled ? 'bg-emerald-500' : 'bg-muted-foreground/30'
              }`}
            >
              <span className={`pointer-events-none inline-block size-5 transform rounded-full bg-white shadow-lg ring-0 transition-transform duration-200 ${poolEnabled ? 'translate-x-5' : 'translate-x-0'}`} />
            </button>
          </div>

          {selected.size > 0 && (
            <button
              onClick={handleBatchDelete}
              className="flex items-center gap-2 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm font-semibold text-destructive transition-colors hover:bg-destructive/20"
            >
              <Trash2 className="size-4" />
              {t('proxies.deleteSelected', { count: selected.size })}
            </button>
          )}

          {proxies.length > 0 && (
            <button
              onClick={handleTestAll}
              disabled={testAllLoading}
              className="flex items-center gap-2 rounded-md border border-border px-3 py-2 text-sm font-semibold text-foreground transition-colors hover:bg-muted/50 disabled:opacity-50"
            >
              {testAllLoading ? <Loader2 className="size-4 animate-spin" /> : <Zap className="size-4" />}
              {testAllLoading
                ? t('proxies.testingAllProgress', { done: testAllDone, total: proxies.length, failed: testAllFailed })
                : t('proxies.testAll')}
            </button>
          )}

          <button
            onClick={() => setShowAdd(!showAdd)}
            className="flex items-center gap-2 rounded-md bg-primary px-3 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition-colors hover:bg-primary/90"
          >
            <Plus className="size-4" />
            {t('proxies.addProxy')}
          </button>
        </div>
      </div>

      {/* Add Panel */}
      {showAdd && (
        <Card className="py-0">
          <CardContent className="p-6 space-y-4">
            <h4 className="text-base font-semibold text-foreground">{t('proxies.addProxyTitle')}</h4>
            <p className="text-sm text-muted-foreground">
              {t('proxies.addProxyDesc')}
            </p>
            <textarea
              value={addInput}
              onChange={e => {
                setAddInput(e.target.value)
                if (addError) setAddError('')
              }}
              placeholder={"http://user:pass@ip:port\nsocks5://ip:port"}
              className="w-full h-32 px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground resize-none outline-none focus:ring-2 focus:ring-primary/30 font-mono"
            />
            {addError && (
              <div className="flex items-center gap-2 rounded-md border border-destructive/20 bg-destructive/10 px-3 py-2 text-sm font-medium text-destructive">
                <AlertTriangle className="size-4 shrink-0" />
                {addError}
              </div>
            )}
            <div className="flex items-center gap-3">
              <input
                type="text"
                value={addLabel}
                onChange={e => setAddLabel(e.target.value)}
                placeholder={t('proxies.labelPlaceholder')}
                className="flex-1 px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground outline-none focus:ring-2 focus:ring-primary/30"
              />
              <button
                onClick={handleAdd}
                disabled={addLoading || !addInput.trim()}
                className="px-5 py-2 rounded-md text-sm font-semibold bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 shadow-sm"
              >
                {addLoading ? t('proxies.adding') : t('proxies.confirmAdd')}
              </button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <Card className="py-0">
          <CardContent className="p-4 text-center">
            <div className="text-2xl font-bold text-foreground">{proxyCounts.total}</div>
            <div className="text-xs text-muted-foreground mt-1">{t('proxies.totalProxies')}</div>
          </CardContent>
        </Card>
        <Card className="py-0">
          <CardContent className="p-4 text-center">
            <div className="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{proxyCounts.enabled}</div>
            <div className="text-xs text-muted-foreground mt-1">{t('proxies.enabledCount')}</div>
          </CardContent>
        </Card>
        <Card className="py-0">
          <CardContent className="p-4 text-center">
            <div className={`text-2xl font-bold ${poolEnabled ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground'}`}>
              {poolEnabled ? t('proxies.roundRobin') : t('proxies.off')}
            </div>
            <div className="text-xs text-muted-foreground mt-1">{t('proxies.poolStatus')}</div>
          </CardContent>
        </Card>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {filterOptions.map(option => (
          <button
            key={option.value}
            onClick={() => {
              setFilter(option.value)
              setPage(1)
            }}
            className={`inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm font-semibold transition-colors ${
              filter === option.value
                ? 'border-primary/40 bg-primary/10 text-primary'
                : 'border-border text-muted-foreground hover:bg-muted/50 hover:text-foreground'
            }`}
          >
            <span>{option.label}</span>
            <span className="font-mono text-xs">{option.count}</span>
          </button>
        ))}
      </div>

      {/* Table */}
      <Card className="py-0">
        <CardContent className="p-0">
          {loading ? (
            <div className="flex justify-center items-center py-16">
              <Loader2 className="size-6 animate-spin text-primary" />
            </div>
          ) : loadError ? (
            <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
              <AlertTriangle className="size-10 text-destructive" />
              <div>
                <p className="text-sm font-semibold text-foreground">{t('common.loadFailed')}</p>
                <p className="mt-1 text-xs text-muted-foreground">{loadError}</p>
              </div>
              <button
                onClick={() => {
                  setLoading(true)
                  void reload()
                }}
                className="rounded-md border border-border px-3 py-1.5 text-sm font-semibold text-foreground transition-colors hover:bg-muted/50"
              >
                {t('common.retry')}
              </button>
            </div>
          ) : filteredProxies.length === 0 ? (
            <div className="text-center py-16 text-muted-foreground">
              <Globe className="size-12 mx-auto mb-3 opacity-30" />
              <p className="text-sm font-medium">{t('proxies.noProxies')}</p>
              <p className="text-xs mt-1">{t('proxies.noProxiesDesc')}</p>
            </div>
          ) : (
            <>
              <div className="data-table-shell">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border text-left text-muted-foreground">
                      <th className="p-3 w-10">
                        <input type="checkbox" checked={allSelected} onChange={toggleSelectAll} className="size-4 rounded" />
                      </th>
                      <th className="p-3 font-semibold">{t('proxies.colUrl')}</th>
                      <th className="p-3 font-semibold">{t('proxies.colStatus')}</th>
                      <th className="p-3 font-semibold">{t('proxies.colLocation')}</th>
                      <th className="p-3 font-semibold">{t('proxies.colIp')}</th>
                      <th className="p-3 font-semibold">{t('proxies.colLatency')}</th>
                      <th className="p-3 font-semibold text-right">{t('proxies.colActions')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {pagedProxies.map(p => {
                      const isTesting = testingIds.has(p.id)
                      return (
                        <tr key={p.id} className="border-b border-border/50 hover:bg-muted/30 transition-colors">
                          <td className="p-3">
                            <input
                              type="checkbox"
                              checked={selected.has(p.id)}
                              onChange={() => {
                                const next = new Set(selected)
                                if (next.has(p.id)) next.delete(p.id)
                                else next.add(p.id)
                                setSelected(next)
                              }}
                              className="size-4 rounded"
                            />
                          </td>
                          <td className="p-3 max-w-[380px]">
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => {
                                  setRevealedIds(prev => {
                                    const next = new Set(prev)
                                    if (next.has(p.id)) next.delete(p.id)
                                    else next.add(p.id)
                                    return next
                                  })
                                }}
                                className="shrink-0 flex items-center justify-center size-6 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-all"
                                title={revealedIds.has(p.id) ? 'Hide' : 'Show'}
                              >
                                {revealedIds.has(p.id) ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                              </button>
                              <span className="font-mono text-[13px] font-medium break-all text-foreground">
                                {revealedIds.has(p.id) ? p.url : maskUrl(p.url)}
                              </span>
                            </div>
                          </td>
                          <td className="p-3">
                            <button
                              onClick={() => handleToggle(p)}
                              className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold transition-all ${
                                p.enabled
                                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
                                  : 'bg-muted/50 text-muted-foreground border border-border'
                              }`}
                            >
                              <span className={`size-1.5 rounded-full ${p.enabled ? 'bg-emerald-500' : 'bg-muted-foreground/50'}`} />
                              {p.enabled ? t('proxies.enabled') : t('proxies.disabled')}
                            </button>
                          </td>
                          {/* Location */}
                          <td className="p-3">
                            {isTesting ? (
                              <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
                            ) : p.test_location ? (
                              <div className="flex items-center gap-1 text-xs font-medium text-foreground whitespace-nowrap">
                                <MapPin className="size-3 text-primary shrink-0" />
                                {p.test_location}
                              </div>
                            ) : (
                              <span className="text-xs text-muted-foreground">-</span>
                            )}
                          </td>
                          {/* IP */}
                          <td className="p-3">
                            {p.test_ip ? (
                              <span className="text-[13px] font-mono font-medium text-foreground whitespace-nowrap">{p.test_ip}</span>
                            ) : (
                              <span className="text-xs text-muted-foreground">-</span>
                            )}
                          </td>
                          {/* Latency */}
                          <td className="p-3">
                            {p.test_latency_ms > 0 ? (
                              <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-bold ${latencyColor(p.test_latency_ms)} ${latencyBg(p.test_latency_ms)}`}>
                                {p.test_latency_ms}ms
                              </span>
                            ) : (
                              <span className="text-xs text-muted-foreground">-</span>
                            )}
                          </td>
                          <td className="p-3">
                            <div className="flex items-center gap-1.5 justify-end">
                              <button
                                onClick={() => handleTest(p)}
                                disabled={isTesting}
                                className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium border border-border text-foreground hover:bg-muted/50 transition-all disabled:opacity-50"
                                title={t('proxies.testProxy')}
                              >
                                {isTesting ? <Loader2 className="size-3.5 animate-spin" /> : <Play className="size-3.5" />}
                                {t('proxies.test')}
                              </button>
                              <button
                                onClick={() => handleDelete(p.id)}
                                className="flex items-center justify-center size-7 rounded-lg text-destructive hover:bg-destructive/10 transition-all"
                                title={t('common.delete')}
                              >
                                <Trash2 className="size-3.5" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between px-4 py-3 border-t border-border">
                  <span className="text-xs text-muted-foreground">
                    {t('proxies.pagination', { total: filteredProxies.length, page, totalPages })}
                  </span>
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => setPage(p => Math.max(1, p - 1))}
                      disabled={page <= 1}
                      className="flex items-center justify-center size-8 rounded-lg border border-border text-foreground hover:bg-muted/50 transition-all disabled:opacity-30 disabled:cursor-not-allowed"
                    >
                      <ChevronLeft className="size-4" />
                    </button>
                    {Array.from({ length: totalPages }, (_, i) => i + 1).map(n => (
                      <button
                        key={n}
                        onClick={() => setPage(n)}
                        className={`flex items-center justify-center size-8 rounded-lg text-xs font-medium transition-all ${
                          n === page
                            ? 'bg-primary text-primary-foreground shadow-sm'
                            : 'border border-border text-foreground hover:bg-muted/50'
                        }`}
                      >
                        {n}
                      </button>
                    ))}
                    <button
                      onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                      disabled={page >= totalPages}
                      className="flex items-center justify-center size-8 rounded-lg border border-border text-foreground hover:bg-muted/50 transition-all disabled:opacity-30 disabled:cursor-not-allowed"
                    >
                      <ChevronRight className="size-4" />
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
