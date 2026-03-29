import type { ReactNode } from 'react'
import { useCallback, useEffect, useRef, useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import DashboardUsageCharts, { getTimeRangeISO, getBucketConfig } from '../components/DashboardUsageCharts'
import type { TimeRangeKey } from '../components/DashboardUsageCharts'
import PageHeader from '../components/PageHeader'
import StateShell from '../components/StateShell'
import StatCard from '../components/StatCard'
import type { StatsResponse, UsageStats, ChartAggregation } from '../types'
import { useDataLoader } from '../hooks/useDataLoader'
import { Card, CardContent } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { Skeleton } from '@/components/ui/skeleton'
import { Users, CheckCircle, XCircle, Activity, Zap, Clock, AlertTriangle, BarChart3, Database } from 'lucide-react'

const DASHBOARD_REFRESH_INTERVAL_MS = 15_000

// 骨架屏组件 - 统计卡片
function StatCardSkeleton() {
  return (
    <Card className="py-0">
      <CardContent className="flex flex-col justify-between gap-2 p-4">
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0 flex-1">
            <Skeleton className="h-3 w-20 mb-2" />
            <Skeleton className="h-8 w-16" />
          </div>
          <Skeleton className="size-10 shrink-0 rounded-xl" />
        </div>
      </CardContent>
    </Card>
  )
}

// 骨架屏组件 - 使用统计
function UsageStatsSkeleton() {
  return (
    <Card>
      <CardContent className="p-6">
        <Skeleton className="h-5 w-32 mb-4" />
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {Array.from({ length: 7 }).map((_, i) => (
            <div key={i} className="flex items-center gap-3 p-4 rounded-xl bg-muted/30">
              <Skeleton className="size-10 rounded-lg shrink-0" />
              <div className="flex-1 min-w-0">
                <Skeleton className="h-3 w-24 mb-1" />
                <Skeleton className="h-5 w-16" />
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

// 错误状态组件
function DashboardError({ onRetry, error }: { onRetry: () => void; error: string | null }) {
  const { t } = useTranslation()
  return (
    <div className="space-y-6">
      <PageHeader
        title={t('dashboard.title')}
        description={t('dashboard.description')}
      />
      <StateShell
        variant="section"
        error={error}
        onRetry={onRetry}
        errorTitle={t('dashboard.errorTitle')}
      >
        <></>
      </StateShell>
    </div>
  )
}

// 统计项组件
function StatItem({ icon, iconBg, label, value, tooltip }: { icon: ReactNode; iconBg: string; label: string; value: string; tooltip?: string }) {
  const content = (
    <div className="flex items-center gap-3 p-4 rounded-xl bg-muted/50 transition-colors hover:bg-muted/70">
      <div className={`flex items-center justify-center size-10 rounded-lg shrink-0 ${iconBg}`}>
        {icon}
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-xs text-muted-foreground truncate">{label}</div>
        <div className="text-lg font-bold truncate">{value}</div>
      </div>
    </div>
  )

  if (tooltip) {
    return (
      <TooltipProvider delayDuration={200}>
        <Tooltip>
          <TooltipTrigger asChild>
            {content}
          </TooltipTrigger>
          <TooltipContent>
            <p>{tooltip}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  return content
}

export default function Dashboard() {
  const { t } = useTranslation()
  const [timeRange, setTimeRange] = useState<TimeRangeKey>('1h')
  const [chartData, setChartData] = useState<ChartAggregation | null>(null)
  const [chartRefreshedAt, setChartRefreshedAt] = useState<number | null>(null)
  const [chartLoading, setChartLoading] = useState(true)
  const chartAbort = useRef<AbortController | null>(null)
  const visibilityRef = useRef(true)

  // 仅加载轻量级统计数据（秒级响应）
  const loadDashboardStats = useCallback(async () => {
    const [stats, usageStats] = await Promise.all([
      api.getStats(),
      api.getUsageStats(),
    ])
    return { stats, usageStats }
  }, [])

  const { data, loading, error, reload, reloadSilently } = useDataLoader<{
    stats: StatsResponse | null
    usageStats: UsageStats | null
  }>({
    initialData: { stats: null, usageStats: null },
    load: loadDashboardStats,
  })

  // 加载服务端聚合的图表数据（12~48 个聚合点，非原始行）
  const loadChartData = useCallback(async () => {
    chartAbort.current?.abort()
    const controller = new AbortController()
    chartAbort.current = controller
    setChartLoading(true)
    try {
      const { start, end } = getTimeRangeISO(timeRange)
      const { bucketMinutes } = getBucketConfig(timeRange)
      const res = await api.getChartData({ start, end, bucketMinutes })
      if (!controller.signal.aborted) {
        setChartData(res)
        setChartRefreshedAt(Date.now())
      }
    } catch {
      // 静默容错
    } finally {
      if (!controller.signal.aborted) {
        setChartLoading(false)
      }
    }
  }, [timeRange])

  // 首次加载 + timeRange 变更时重新拉取图表数据
  useEffect(() => {
    void loadChartData()
  }, [loadChartData])

  // 监听页面可见性变化
  useEffect(() => {
    const handleVisibilityChange = () => {
      visibilityRef.current = document.visibilityState === 'visible'
    }
    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
  }, [])

  // 仅在 1h（实时）模式下启用自动刷新
  useEffect(() => {
    if (timeRange !== '1h') return

    const timer = window.setInterval(() => {
      if (!visibilityRef.current) return
      void reloadSilently()
      void loadChartData()
    }, DASHBOARD_REFRESH_INTERVAL_MS)

    return () => window.clearInterval(timer)
  }, [reloadSilently, timeRange, loadChartData])

  // 清理 AbortController
  useEffect(() => {
    return () => {
      chartAbort.current?.abort()
    }
  }, [])

  const { stats, usageStats } = data

  // 计算统计数据
  const statsData = useMemo(() => ({
    total: stats?.total ?? 0,
    available: stats?.available ?? 0,
    errorCount: stats?.error ?? 0,
    todayRequests: stats?.today_requests ?? 0,
    availableRate: stats?.total ? Math.round((stats.available / stats.total) * 100) : 0,
  }), [stats])

  const icons = useMemo(() => ({
    total: <Users className="size-[22px]" />,
    available: <CheckCircle className="size-[22px]" />,
    error: <XCircle className="size-[22px]" />,
    requests: <Activity className="size-[22px]" />,
  }), [])

  // 处理重试
  const handleRetry = useCallback(() => {
    void reload()
    void loadChartData()
  }, [reload, loadChartData])

  // 如果页面加载出错，显示错误状态
  if (error && loading) {
    return <DashboardError onRetry={handleRetry} error={error} />
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('dashboard.title')}
        description={t('dashboard.description')}
        onRefresh={handleRetry}
      />

      {/* Account status - 响应式网格布局 */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {loading ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard
              icon={icons.total}
              iconClass="blue"
              label={t('dashboard.totalAccounts')}
              value={statsData.total}
            />
            <StatCard
              icon={icons.available}
              iconClass="green"
              label={t('dashboard.available')}
              value={statsData.available}
              sub={t('dashboard.availableRate', { rate: statsData.availableRate })}
            />
            <StatCard
              icon={icons.error}
              iconClass="red"
              label={t('dashboard.error')}
              value={statsData.errorCount}
            />
            <StatCard
              icon={icons.requests}
              iconClass="purple"
              label={t('dashboard.todayRequests')}
              value={statsData.todayRequests}
            />
          </>
        )}
      </div>

      {/* Usage stats - 响应式布局 */}
      {loading ? (
        <UsageStatsSkeleton />
      ) : usageStats ? (
        <div className="space-y-4">
          <Card>
            <CardContent className="p-4 sm:p-6">
              <h3 className="text-base font-semibold text-foreground mb-4">{t('dashboard.usageStats')}</h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 sm:gap-4">
                <StatItem
                  icon={<BarChart3 className="size-5" />}
                  iconBg="bg-blue-500/10 text-blue-500 dark:text-blue-400"
                  label={t('dashboard.totalRequests')}
                  value={usageStats.total_requests.toLocaleString()}
                  tooltip={t('dashboard.totalRequests')}
                />
                <StatItem
                  icon={<Zap className="size-5" />}
                  iconBg="bg-purple-500/10 text-purple-500 dark:text-purple-400"
                  label={t('dashboard.totalTokens')}
                  value={usageStats.total_tokens.toLocaleString()}
                  tooltip={t('dashboard.totalTokens')}
                />
                <StatItem
                  icon={<Zap className="size-5" />}
                  iconBg="bg-emerald-500/10 text-emerald-500 dark:text-emerald-400"
                  label={t('dashboard.todayTokens')}
                  value={usageStats.today_tokens.toLocaleString()}
                  tooltip={t('dashboard.todayTokens')}
                />
                <StatItem
                  icon={<Database className="size-5" />}
                  iconBg="bg-indigo-500/10 text-indigo-500 dark:text-indigo-400"
                  label={t('dashboard.cachedTokens')}
                  value={usageStats.total_cached_tokens.toLocaleString()}
                  tooltip={t('dashboard.cachedTokens')}
                />
                <StatItem
                  icon={<Activity className="size-5" />}
                  iconBg="bg-amber-500/10 text-amber-500 dark:text-amber-400"
                  label={t('dashboard.rpmTpm')}
                  value={`${usageStats.rpm} / ${usageStats.tpm.toLocaleString()}`}
                  tooltip={`${t('dashboard.rpmTpm')}`}
                />
                <StatItem
                  icon={<Clock className="size-5" />}
                  iconBg="bg-cyan-500/10 text-cyan-500 dark:text-cyan-400"
                  label={t('dashboard.avgLatency')}
                  value={usageStats.avg_duration_ms > 1000 ? `${(usageStats.avg_duration_ms / 1000).toFixed(1)}s` : `${Math.round(usageStats.avg_duration_ms)}ms`}
                  tooltip={t('dashboard.avgLatency')}
                />
                <StatItem
                  icon={<AlertTriangle className="size-5" />}
                  iconBg="bg-red-500/10 text-red-500 dark:text-red-400"
                  label={t('dashboard.todayErrorRate')}
                  value={`${usageStats.error_rate.toFixed(1)}%`}
                  tooltip={t('dashboard.todayErrorRate')}
                />
              </div>
            </CardContent>
          </Card>

          <DashboardUsageCharts
            chartData={chartData}
            refreshedAt={chartRefreshedAt}
            refreshIntervalMs={DASHBOARD_REFRESH_INTERVAL_MS}
            timeRange={timeRange}
            onTimeRangeChange={setTimeRange}
            loading={chartLoading}
          />
        </div>
      ) : null}
    </div>
  )
}
