import { useCallback, useEffect, useMemo, useState } from 'react'
import { Activity, AlertTriangle, BarChart3, Clock3, Cpu, Database, HardDrive, RefreshCw, Server, Users, Zap } from 'lucide-react'
import { api } from '../api'
import PageHeader from '../components/PageHeader'
import StateShell from '../components/StateShell'
import { useDataLoader } from '../hooks/useDataLoader'
import StatusBadge from '../components/StatusBadge'
import type { AccountRow, OpsOverviewResponse } from '../types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Select } from '@/components/ui/select'

type MetricTone = 'normal' | 'warning' | 'danger' | 'info'

export default function Operations() {
  const [tierFilter, setTierFilter] = useState('all')
  const [sortBy, setSortBy] = useState('risk')
  const loadOperationsData = useCallback(async () => {
    const [overview, accountsResponse] = await Promise.all([
      api.getOpsOverview(),
      api.getAccounts(),
    ])

    return {
      overview,
      accounts: accountsResponse.accounts ?? [],
    }
  }, [])

  const { data, loading, error, reload, reloadSilently } = useDataLoader<{
    overview: OpsOverviewResponse | null
    accounts: AccountRow[]
  }>({
    initialData: {
      overview: null,
      accounts: [],
    },
    load: loadOperationsData,
  })

  useEffect(() => {
    const timer = window.setInterval(() => {
      void reloadSilently()
    }, 15000)

    return () => window.clearInterval(timer)
  }, [reloadSilently])

  const overview = data.overview
  const accounts = data.accounts
  const updatedLabel = overview?.updated_at ? formatTimeLabel(overview.updated_at) : '--:--:--'
  const schedulerCounts = useMemo(() => ({
    healthy: accounts.filter((account) => account.health_tier === 'healthy').length,
    warm: accounts.filter((account) => account.health_tier === 'warm').length,
    risky: accounts.filter((account) => account.health_tier === 'risky').length,
    banned: accounts.filter((account) => account.health_tier === 'banned' || account.status === 'unauthorized').length,
  }), [accounts])
  const recentIssueCounts = useMemo(() => {
    const now = Date.now()
    const withinWindow = (iso?: string, minutes = 60) => {
      if (!iso) return false
      const ts = new Date(iso).getTime()
      if (Number.isNaN(ts)) return false
      return now-ts <= minutes*60*1000
    }

    return {
      unauthorized24h: accounts.filter((account) => withinWindow(account.last_unauthorized_at, 24 * 60)).length,
      rateLimited1h: accounts.filter((account) => withinWindow(account.last_rate_limited_at, 60)).length,
      timeout15m: accounts.filter((account) => withinWindow(account.last_timeout_at, 15)).length,
    }
  }, [accounts])
  const spotlightAccounts = useMemo(() => {
    const priority = (account: AccountRow) => {
      if (account.health_tier === 'banned' || account.status === 'unauthorized') return 3
      if (account.health_tier === 'risky') return 2
      if (account.health_tier === 'warm') return 1
      return 0
    }

    return [...accounts]
      .filter((account) => {
        if (tierFilter === 'all') {
          return priority(account) > 0
        }
        if (tierFilter === 'banned') {
          return account.health_tier === 'banned' || account.status === 'unauthorized'
        }
        return account.health_tier === tierFilter
      })
      .sort((left, right) => {
        switch (sortBy) {
          case 'score_asc':
            return (left.scheduler_score ?? 0) - (right.scheduler_score ?? 0)
          case 'usage_desc':
            return (right.usage_percent_7d ?? -1) - (left.usage_percent_7d ?? -1)
          case 'latency_penalty':
            return (right.scheduler_breakdown?.latency_penalty ?? 0) - (left.scheduler_breakdown?.latency_penalty ?? 0)
          case 'unauthorized':
            return Number(Boolean(right.last_unauthorized_at)) - Number(Boolean(left.last_unauthorized_at))
          case 'risk':
          default: {
            const priorityDiff = priority(right) - priority(left)
            if (priorityDiff !== 0) return priorityDiff
            return (left.scheduler_score ?? 0) - (right.scheduler_score ?? 0)
          }
        }
      })
      .slice(0, 8)
  }, [accounts, tierFilter, sortBy])

  return (
    <StateShell
      variant="page"
      loading={loading}
      error={error}
      onRetry={() => void reload()}
      loadingTitle="正在加载系统运维"
      loadingDescription="服务状态、连接池和实时流量正在同步。"
      errorTitle="系统运维页加载失败"
    >
      <>
        <PageHeader
          title="系统运维"
          description="查看服务运行状态、连接池压力和实时流量表现。"
          actions={
            <div className="flex items-center gap-3 max-sm:w-full max-sm:justify-between">
              <span className="text-sm text-muted-foreground">最后更新时间：{updatedLabel}</span>
              <Button variant="outline" onClick={() => void reload()}>
                <RefreshCw className="size-3.5" />
                刷新
              </Button>
            </div>
          }
        />

        {overview ? (
          <>
            <div className="grid grid-cols-[repeat(auto-fit,minmax(220px,1fr))] gap-4 mb-6">
              <SummaryPill label="运行时长" value={formatUptime(overview.uptime_seconds)} />
              <SummaryPill label="账号池" value={`${overview.runtime.available_accounts} / ${overview.runtime.total_accounts}`} />
              <SummaryPill label="今日请求" value={formatNumber(overview.traffic.today_requests)} />
              <SummaryPill label="今日错误率" value={`${overview.traffic.error_rate.toFixed(1)}%`} />
            </div>

            <Card className="mb-6">
              <CardContent className="p-6">
                <div className="flex items-center justify-between gap-4 max-sm:flex-col max-sm:items-start">
                  <div>
                    <h3 className="text-base font-semibold text-foreground">调度全局视图</h3>
                    <p className="mt-1 text-sm text-muted-foreground">从系统维度查看当前号池健康分层和高风险账号分布。</p>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <SchedulerPill label="Healthy" value={schedulerCounts.healthy} tone="success" />
                    <SchedulerPill label="Warm" value={schedulerCounts.warm} tone="warning" />
                    <SchedulerPill label="Risky" value={schedulerCounts.risky} tone="danger" />
                    <SchedulerPill label="Banned" value={schedulerCounts.banned} tone="neutral" />
                  </div>
                </div>

                <div className="mt-4 grid gap-3 md:grid-cols-3">
                  <MiniOpsCard
                    label="最近 401"
                    value={formatNumber(recentIssueCounts.unauthorized24h)}
                    sub="24 小时内被判定 unauthorized 的账号数"
                    tone="danger"
                  />
                  <MiniOpsCard
                    label="最近 429"
                    value={formatNumber(recentIssueCounts.rateLimited1h)}
                    sub="1 小时内触发限流的账号数"
                    tone="warning"
                  />
                  <MiniOpsCard
                    label="最近 Timeout"
                    value={formatNumber(recentIssueCounts.timeout15m)}
                    sub="15 分钟内出现超时的账号数"
                    tone="neutral"
                  />
                </div>

                <div className="mt-5 flex flex-wrap items-center gap-3 rounded-2xl border border-border bg-white/45 px-4 py-3">
                  <span className="text-[12px] font-semibold text-muted-foreground">筛选</span>
                  <div className="w-[180px]">
                    <Select
                      value={tierFilter}
                      onValueChange={setTierFilter}
                      options={[
                        { label: '全部风险账号', value: 'all' },
                        { label: 'Warm', value: 'warm' },
                        { label: 'Risky', value: 'risky' },
                        { label: 'Banned', value: 'banned' },
                      ]}
                    />
                  </div>
                  <span className="text-[12px] font-semibold text-muted-foreground">排序</span>
                  <div className="w-[200px]">
                    <Select
                      value={sortBy}
                      onValueChange={setSortBy}
                      options={[
                        { label: '风险优先', value: 'risk' },
                        { label: '分数升序', value: 'score_asc' },
                        { label: '7d 用量高优先', value: 'usage_desc' },
                        { label: '延迟惩罚高优先', value: 'latency_penalty' },
                        { label: '401 近期优先', value: 'unauthorized' },
                      ]}
                    />
                  </div>
                </div>

                <div className="mt-5 grid gap-3 md:grid-cols-2">
                  {spotlightAccounts.length > 0 ? (
                    spotlightAccounts.map((account) => (
                      <div key={account.id} className="rounded-2xl border border-border bg-white/50 px-4 py-3">
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0">
                            <div className="truncate text-[14px] font-semibold text-foreground">
                              {account.email || `ID ${account.id}`}
                            </div>
                            <div className="mt-1 text-[12px] text-muted-foreground">
                              分 {Math.round(account.scheduler_score ?? 0)} · 并发 {account.dynamic_concurrency_limit ?? '-'} · 套餐 {account.plan_type || '-'}
                            </div>
                          </div>
                          <StatusBadge status={account.status} />
                        </div>
                        <div className="mt-3 flex flex-wrap items-center gap-2">
                          <Badge variant="outline" className={getHealthTierClassName(account.health_tier)}>
                            {formatHealthTier(account.health_tier)}
                          </Badge>
                          {account.usage_percent_7d !== null && account.usage_percent_7d !== undefined ? (
                            <Badge variant="outline" className="text-[12px]">
                              7d {account.usage_percent_7d.toFixed(1)}%
                            </Badge>
                          ) : null}
                        </div>
                        <div className="mt-3 flex flex-wrap items-center gap-2">
                          {buildScoreReasonTags(account).map((tag) => (
                            <Badge key={tag.label} variant="outline" className={tag.className}>
                              {tag.label}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className="rounded-2xl border border-border bg-white/40 px-4 py-4 text-sm text-muted-foreground">
                      当前没有需要重点关注的风险账号，号池整体处于稳定状态。
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-6">
                <div className="mb-5 flex items-center justify-between gap-4">
                  <div>
                    <h3 className="text-base font-semibold text-foreground">系统概览</h3>
                    <p className="mt-1 text-sm text-muted-foreground">按 15 秒自动刷新，适合快速查看当前服务健康度。</p>
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
                  <OpsMetricCard
                    label="CPU"
                    value={`${overview.cpu.percent.toFixed(1)}%`}
                    sub={`核心数 ${overview.cpu.cores} 核`}
                    icon={<Cpu className="size-5" />}
                    tone={getPercentTone(overview.cpu.percent, 70, 90)}
                  />
                  <OpsMetricCard
                    label="内存"
                    value={`${overview.memory.percent.toFixed(1)}%`}
                    sub={`使用 ${formatBytes(overview.memory.used_bytes)} / ${formatBytes(overview.memory.total_bytes)}`}
                    icon={<HardDrive className="size-5" />}
                    tone={getPercentTone(overview.memory.percent, 75, 90)}
                  />
                  <OpsMetricCard
                    label="PostgreSQL"
                    value={`${overview.postgres.usage_percent.toFixed(1)}%`}
                    sub={`连接 ${overview.postgres.open} / ${overview.postgres.max_open || '∞'}`}
                    icon={<Database className="size-5" />}
                    tone={overview.postgres.healthy ? getPercentTone(overview.postgres.usage_percent, 75, 90) : 'danger'}
                  />
                  <OpsMetricCard
                    label="Redis"
                    value={`${overview.redis.usage_percent.toFixed(1)}%`}
                    sub={`连接 ${overview.redis.total_conns} / ${overview.redis.pool_size || '-'}`}
                    icon={<Server className="size-5" />}
                    tone={overview.redis.healthy ? getPercentTone(overview.redis.usage_percent, 70, 90) : 'danger'}
                  />
                  <OpsMetricCard
                    label="当前请求"
                    value={formatNumber(overview.requests.active)}
                    sub={`运行期累计 ${formatNumber(overview.requests.total)}`}
                    icon={<Activity className="size-5" />}
                    tone={overview.requests.active >= 20 ? 'warning' : 'normal'}
                  />
                  <OpsMetricCard
                    label="协程"
                    value={formatNumber(overview.runtime.goroutines)}
                    sub={`账号池 ${overview.runtime.available_accounts} / ${overview.runtime.total_accounts}`}
                    icon={<Users className="size-5" />}
                    tone={overview.runtime.goroutines >= 500 ? 'danger' : overview.runtime.goroutines >= 200 ? 'warning' : 'normal'}
                  />
                  <OpsMetricCard
                    label="QPS"
                    value={overview.traffic.qps.toFixed(1)}
                    sub={`峰值 ${overview.traffic.qps_peak.toFixed(1)}`}
                    icon={<BarChart3 className="size-5" />}
                    tone="info"
                  />
                  <OpsMetricCard
                    label="TPS"
                    value={formatNumber(Math.round(overview.traffic.tps))}
                    sub={`峰值 ${formatNumber(Math.round(overview.traffic.tps_peak))}`}
                    icon={<Zap className="size-5" />}
                    tone="info"
                  />
                  <OpsMetricCard
                    label="RPM"
                    value={formatNumber(Math.round(overview.traffic.rpm))}
                    sub={overview.traffic.rpm_limit > 0 ? `限额 ${formatNumber(overview.traffic.rpm_limit)}` : '未开启全局限流'}
                    icon={<Clock3 className="size-5" />}
                    tone={overview.traffic.rpm_limit > 0 && overview.traffic.rpm >= overview.traffic.rpm_limit * 0.8 ? 'warning' : 'normal'}
                  />
                  <OpsMetricCard
                    label="TPM"
                    value={formatNumber(Math.round(overview.traffic.tpm))}
                    sub={`今日 ${formatNumber(overview.traffic.today_tokens)}`}
                    icon={<AlertTriangle className="size-5" />}
                    tone={overview.traffic.error_rate >= 5 ? 'warning' : 'normal'}
                  />
                </div>
              </CardContent>
            </Card>
          </>
        ) : null}
      </>
    </StateShell>
  )
}

function OpsMetricCard({
  label,
  value,
  sub,
  icon,
  tone,
}: {
  label: string
  value: string
  sub: string
  icon: React.ReactNode
  tone: MetricTone
}) {
  const toneStyle = {
    normal: {
      badge: 'bg-[hsl(var(--success-bg))] text-[hsl(var(--success))]',
      dot: 'bg-emerald-500',
      icon: 'bg-[hsl(var(--success-bg))] text-[hsl(var(--success))]',
      label: '正常',
    },
    warning: {
      badge: 'bg-amber-500/10 text-amber-600',
      dot: 'bg-amber-500',
      icon: 'bg-amber-500/10 text-amber-600',
      label: '偏高',
    },
    danger: {
      badge: 'bg-destructive/10 text-destructive',
      dot: 'bg-destructive',
      icon: 'bg-destructive/10 text-destructive',
      label: '异常',
    },
    info: {
      badge: 'bg-primary/10 text-primary',
      dot: 'bg-primary',
      icon: 'bg-primary/10 text-primary',
      label: '实时',
    },
  }[tone]

  return (
    <Card className="py-0 transition-all duration-150 hover:-translate-y-0.5 hover:shadow-md">
      <CardContent className="p-4">
        <div className="flex items-center justify-between gap-3">
          <span className="text-[13px] font-semibold text-muted-foreground">{label}</span>
          <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-[12px] font-semibold ${toneStyle.badge}`}>
            <span className={`size-2 rounded-full ${toneStyle.dot}`} />
            {toneStyle.label}
          </span>
        </div>

        <div className="mt-5 flex items-end justify-between gap-3">
          <div className="min-w-0">
            <div className="text-[34px] font-bold leading-none tracking-tighter text-foreground">{value}</div>
            <div className="mt-3 text-[13px] leading-relaxed text-muted-foreground">{sub}</div>
          </div>
          <div className={`flex size-11 shrink-0 items-center justify-center rounded-2xl ${toneStyle.icon}`}>
            {icon}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function SummaryPill({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-border bg-white/65 px-4 py-3 shadow-[inset_0_1px_0_rgba(255,255,255,0.7)]">
      <div className="text-[12px] font-bold tracking-[0.14em] uppercase text-muted-foreground">{label}</div>
      <div className="mt-2 text-[20px] font-bold tracking-tight text-foreground">{value}</div>
    </div>
  )
}

function SchedulerPill({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: 'neutral' | 'success' | 'warning' | 'danger'
}) {
  const toneStyle = {
    neutral: 'bg-slate-500/10 text-slate-600 dark:bg-slate-500/20 dark:text-slate-300',
    success: 'bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-300',
    warning: 'bg-amber-500/10 text-amber-600 dark:bg-amber-500/20 dark:text-amber-300',
    danger: 'bg-red-500/10 text-red-600 dark:bg-red-500/20 dark:text-red-300',
  }[tone]

  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[12px] font-semibold ${toneStyle}`}>
      <span>{label}</span>
      <span>{value}</span>
    </span>
  )
}

function MiniOpsCard({
  label,
  value,
  sub,
  tone,
}: {
  label: string
  value: string
  sub: string
  tone: 'neutral' | 'warning' | 'danger'
}) {
  const toneStyle = {
    neutral: 'bg-slate-500/10 text-slate-600 dark:bg-slate-500/20 dark:text-slate-300',
    warning: 'bg-amber-500/10 text-amber-600 dark:bg-amber-500/20 dark:text-amber-300',
    danger: 'bg-red-500/10 text-red-600 dark:bg-red-500/20 dark:text-red-300',
  }[tone]

  return (
    <div className="rounded-2xl border border-border bg-white/45 px-4 py-4">
      <div className="text-[12px] font-semibold text-muted-foreground">{label}</div>
      <div className="mt-2 text-[28px] font-bold leading-none tracking-tight text-foreground">{value}</div>
      <div className={`mt-3 inline-flex rounded-full px-2.5 py-1 text-[11px] font-semibold ${toneStyle}`}>
        {sub}
      </div>
    </div>
  )
}

function getHealthTierClassName(healthTier?: string) {
  switch (healthTier) {
    case 'healthy':
      return 'border-transparent bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-300'
    case 'warm':
      return 'border-transparent bg-amber-500/10 text-amber-600 dark:bg-amber-500/20 dark:text-amber-300'
    case 'risky':
      return 'border-transparent bg-red-500/10 text-red-600 dark:bg-red-500/20 dark:text-red-300'
    case 'banned':
      return 'border-transparent bg-slate-500/10 text-slate-600 dark:bg-slate-500/20 dark:text-slate-300'
    default:
      return 'border-border text-muted-foreground'
  }
}

function formatHealthTier(healthTier?: string) {
  switch (healthTier) {
    case 'healthy':
      return '健康'
    case 'warm':
      return '预热'
    case 'risky':
      return '风险'
    case 'banned':
      return '隔离'
    default:
      return '未知'
  }
}

function buildScoreReasonTags(account: AccountRow) {
  const breakdown = account.scheduler_breakdown
  if (!breakdown) {
    return []
  }

  const tags: Array<{ label: string; className: string }> = []

  if (breakdown.unauthorized_penalty > 0) {
    tags.push({ label: `401 -${Math.round(breakdown.unauthorized_penalty)}`, className: 'border-transparent bg-red-500/10 text-red-600 dark:bg-red-500/20 dark:text-red-300' })
  }
  if (breakdown.rate_limit_penalty > 0) {
    tags.push({ label: `429 -${Math.round(breakdown.rate_limit_penalty)}`, className: 'border-transparent bg-amber-500/10 text-amber-600 dark:bg-amber-500/20 dark:text-amber-300' })
  }
  if (breakdown.timeout_penalty > 0) {
    tags.push({ label: `超时 -${Math.round(breakdown.timeout_penalty)}`, className: 'border-transparent bg-orange-500/10 text-orange-600 dark:bg-orange-500/20 dark:text-orange-300' })
  }
  if (breakdown.server_penalty > 0) {
    tags.push({ label: `5xx -${Math.round(breakdown.server_penalty)}`, className: 'border-transparent bg-rose-500/10 text-rose-600 dark:bg-rose-500/20 dark:text-rose-300' })
  }
  if (breakdown.failure_penalty > 0) {
    tags.push({ label: `失败串 -${Math.round(breakdown.failure_penalty)}`, className: 'border-transparent bg-slate-500/10 text-slate-600 dark:bg-slate-500/20 dark:text-slate-300' })
  }
  if (breakdown.usage_penalty_7d > 0) {
    tags.push({ label: `7d -${Math.round(breakdown.usage_penalty_7d)}`, className: 'border-transparent bg-fuchsia-500/10 text-fuchsia-600 dark:bg-fuchsia-500/20 dark:text-fuchsia-300' })
  }
  if (breakdown.latency_penalty > 0) {
    tags.push({ label: `延迟 -${Math.round(breakdown.latency_penalty)}`, className: 'border-transparent bg-cyan-500/10 text-cyan-700 dark:bg-cyan-500/20 dark:text-cyan-300' })
  }
  if (breakdown.success_bonus > 0) {
    tags.push({ label: `成功 +${Math.round(breakdown.success_bonus)}`, className: 'border-transparent bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-300' })
  }

  return tags
}

function getPercentTone(value: number, warningThreshold: number, dangerThreshold: number): MetricTone {
  if (value >= dangerThreshold) return 'danger'
  if (value >= warningThreshold) return 'warning'
  return 'normal'
}

function formatBytes(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }
  return `${value.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}

function formatNumber(value: number): string {
  return value.toLocaleString()
}

function formatTimeLabel(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return '--:--:--'
  }
  return date.toLocaleTimeString('zh-CN', {
    hour12: false,
  })
}

function formatUptime(seconds: number): string {
  if (seconds <= 0) return '刚刚启动'

  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) {
    return `${days}天 ${hours}小时`
  }
  if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`
  }
  return `${minutes}分钟`
}
