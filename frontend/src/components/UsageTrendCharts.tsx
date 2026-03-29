import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Area,
  Bar,
  BarChart,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { api } from '../api'
import { Card, CardContent } from '@/components/ui/card'
import type { ChartAggregation, ChartTimelinePoint } from '../types'

export type UsageTimeRange = '1h' | '6h' | '24h' | '7d' | '30d'

interface UsageTrendChartsProps {
  timeRange: UsageTimeRange
  onTimeRangeChange: (range: UsageTimeRange) => void
}

interface TimelineDisplayPoint {
  label: string
  fullLabel: string
  requests: number
  avgLatency: number | null
  inputTokens: number
  outputTokens: number
  reasoningTokens: number
  cachedTokens: number
  errors401: number
  errorRate: number
}

interface ModelRankingPoint {
  model: string
  shortModel: string
  requests: number
  tokens: number
}

interface AccountRankingPoint {
  account: string
  shortAccount: string
  requests: number
  tokens: number
}

const RANGE_OPTIONS: UsageTimeRange[] = ['1h', '6h', '24h', '7d', '30d']

const chartMargin = { top: 8, right: 12, left: -12, bottom: 0 }
const gridColor = 'var(--color-border)'
const axisColor = 'var(--color-muted-foreground)'
const tooltipContentStyle = {
  backgroundColor: 'var(--color-card)',
  border: '1px solid var(--color-border)',
  borderRadius: '16px',
  boxShadow: '0 18px 40px rgba(0, 0, 0, 0.12)',
}
const tooltipLabelStyle = { color: 'var(--color-foreground)', fontWeight: 600 }
const tooltipItemStyle = { color: 'var(--color-foreground)' }

function getBucketConfig(range: UsageTimeRange): { bucketMinutes: number; bucketCount: number } {
  switch (range) {
    case '1h':
      return { bucketMinutes: 5, bucketCount: 12 }
    case '6h':
      return { bucketMinutes: 15, bucketCount: 24 }
    case '24h':
      return { bucketMinutes: 30, bucketCount: 48 }
    case '7d':
      return { bucketMinutes: 360, bucketCount: 28 }
    case '30d':
      return { bucketMinutes: 1440, bucketCount: 30 }
    default:
      return { bucketMinutes: 5, bucketCount: 12 }
  }
}

function toLocalRFC3339(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  const offset = date.getTimezoneOffset()
  const sign = offset <= 0 ? '+' : '-'
  const absOffset = Math.abs(offset)
  const tzH = pad(Math.floor(absOffset / 60))
  const tzM = pad(absOffset % 60)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}${sign}${tzH}:${tzM}`
}

function getTimeRangeISO(range: UsageTimeRange): { start: string; end: string } {
  const now = new Date()
  const end = toLocalRFC3339(now)
  let offsetMs: number
  switch (range) {
    case '1h':
      offsetMs = 60 * 60 * 1000
      break
    case '6h':
      offsetMs = 6 * 60 * 60 * 1000
      break
    case '24h':
      offsetMs = 24 * 60 * 60 * 1000
      break
    case '7d':
      offsetMs = 7 * 24 * 60 * 60 * 1000
      break
    case '30d':
      offsetMs = 30 * 24 * 60 * 60 * 1000
      break
    default:
      offsetMs = 60 * 60 * 1000
  }
  const start = toLocalRFC3339(new Date(now.getTime() - offsetMs))
  return { start, end }
}

function formatMinuteLabel(date: Date): string {
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${hours}:${minutes}`
}

function formatDateLabel(date: Date, bucketMinutes: number): string {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  if (bucketMinutes >= 1440) {
    return `${month}-${day}`
  }
  const hour = String(date.getHours()).padStart(2, '0')
  return `${month}-${day} ${hour}:00`
}

function formatFullLabel(date: Date, bucketMinutes: number): string {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  if (bucketMinutes >= 1440) {
    return `${date.getFullYear()}-${month}-${day}`
  }
  return `${month}-${day} ${hour}:${minute}`
}

function formatCompactNumber(value: number | string): string {
  const numericValue = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(numericValue)) return '0'
  return new Intl.NumberFormat(undefined, { notation: 'compact', maximumFractionDigits: 1 }).format(numericValue)
}

function formatNumber(value: unknown): string {
  const numericValue = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(numericValue)) return '0'
  return numericValue.toLocaleString()
}

function formatDuration(value: unknown): string {
  const numericValue = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(numericValue) || numericValue <= 0) return '-'
  if (numericValue >= 1000) {
    return `${(numericValue / 1000).toFixed(numericValue >= 10000 ? 0 : 1)}s`
  }
  return `${Math.round(numericValue)}ms`
}

function truncateLabel(value: string, maxLength: number): string {
  if (value.length <= maxLength) return value
  return `${value.slice(0, maxLength - 1)}…`
}

export default function UsageTrendCharts({ timeRange, onTimeRangeChange }: UsageTrendChartsProps) {
  const { t } = useTranslation()
  const [chartData, setChartData] = useState<ChartAggregation | null>(null)
  const [loading, setLoading] = useState(false)
  const [accountStats, setAccountStats] = useState<{ account: string; requests: number; tokens: number }[]>([])
  const { bucketMinutes, bucketCount } = getBucketConfig(timeRange)
  const isLive = timeRange === '1h'

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const { start, end } = getTimeRangeISO(timeRange)
      const { bucketMinutes } = getBucketConfig(timeRange)
      const data = await api.getChartData({ start, end, bucketMinutes })
      setChartData(data)

      // 从日志数据聚合账号统计
      const logsResp = await api.getUsageLogsPaged({
        start,
        end,
        page: 1,
        pageSize: 1000,
      })
      const accountMap = new Map<string, { requests: number; tokens: number }>()
      logsResp.logs.forEach((log) => {
        const email = log.account_email || 'unknown'
        const existing = accountMap.get(email) || { requests: 0, tokens: 0 }
        existing.requests += 1
        existing.tokens += log.total_tokens || 0
        accountMap.set(email, existing)
      })
      const stats = Array.from(accountMap.entries())
        .map(([account, data]) => ({ account, ...data }))
        .sort((a, b) => b.requests - a.requests)
        .slice(0, 10)
      setAccountStats(stats)
    } catch (err) {
      console.error('Failed to load chart data:', err)
    } finally {
      setLoading(false)
    }
  }, [timeRange])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  // 自动刷新
  useEffect(() => {
    const timer = setInterval(() => {
      void fetchData()
    }, 30000)
    return () => clearInterval(timer)
  }, [fetchData])

  const displayData = useMemo<{
    timelineData: TimelineDisplayPoint[]
    modelData: ModelRankingPoint[]
    accountData: AccountRankingPoint[]
    totalRequests: number
    totalTokens: number
    avgErrorRate: number
  }>(() => {
    if (!chartData) {
      return {
        timelineData: [],
        modelData: [],
        accountData: [],
        totalRequests: 0,
        totalTokens: 0,
        avgErrorRate: 0,
      }
    }

    const totalRequests = chartData.timeline.reduce((sum, p) => sum + p.requests, 0)
    const totalTokens = chartData.timeline.reduce((sum, p) => sum + p.input_tokens + p.output_tokens, 0)
    const totalErrors = chartData.timeline.reduce((sum, p) => sum + p.errors_401, 0)
    const avgErrorRate = totalRequests > 0 ? (totalErrors / totalRequests) * 100 : 0

    const timelineData: TimelineDisplayPoint[] = chartData.timeline.map((point) => {
      const d = new Date(point.bucket)
      const useFullDate = bucketMinutes >= 360
      const errorRate = point.requests > 0 ? (point.errors_401 / point.requests) * 100 : 0

      return {
        label: useFullDate ? formatDateLabel(d, bucketMinutes) : formatMinuteLabel(d),
        fullLabel: formatFullLabel(d, bucketMinutes),
        requests: point.requests,
        avgLatency: point.avg_latency > 0 ? Math.round(point.avg_latency) : null,
        inputTokens: point.input_tokens,
        outputTokens: point.output_tokens,
        reasoningTokens: point.reasoning_tokens,
        cachedTokens: point.cached_tokens,
        errors401: point.errors_401,
        errorRate: Math.round(errorRate * 10) / 10,
      }
    })

    const modelData: ModelRankingPoint[] = chartData.models
      .slice()
      .sort((a, b) => b.requests - a.requests)
      .slice(0, 8)
      .map((m) => ({
        model: m.model,
        shortModel: truncateLabel(m.model, 22),
        requests: m.requests,
        tokens: m.tokens || 0,
      }))

    const accountData: AccountRankingPoint[] = accountStats.slice(0, 8).map((a) => ({
      account: a.account,
      shortAccount: truncateLabel(a.account, 20),
      requests: a.requests,
      tokens: a.tokens,
    }))

    return { timelineData, modelData, accountData, totalRequests, totalTokens, avgErrorRate }
  }, [chartData, bucketMinutes, accountStats])

  const hasData = displayData.timelineData.length > 0

  return (
    <div className="space-y-4">
      {/* 时间范围选择 */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h3 className="text-base font-semibold text-foreground">{t('usageCharts.title')}</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('usageCharts.description', { count: displayData.totalRequests.toLocaleString() })}
          </p>
          {isLive && (
            <p className="mt-1 text-xs text-muted-foreground">
              {t('usageCharts.liveDesc', { minutes: bucketMinutes })}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {isLive && (
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-300 mr-2">
              <span className="size-2 rounded-full bg-current animate-pulse" />
              <span>{t('dashboard.liveBadge')}</span>
            </div>
          )}
          <div className="inline-flex rounded-lg border border-border bg-muted/50 p-0.5">
            {RANGE_OPTIONS.map((key) => (
              <button
                key={key}
                type="button"
                onClick={() => onTimeRangeChange(key)}
                className={`px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200 ${
                  timeRange === key
                    ? 'bg-background text-foreground shadow-sm border border-border'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {t(`dashboard.timeRange${key.toUpperCase()}`)}
              </button>
            ))}
          </div>
        </div>
      </div>

      {loading && !hasData ? (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {[0, 1, 2, 3, 4, 5].map((i) => (
            <Card key={i} className="py-0">
              <CardContent className="p-6">
                <div className="mb-5 space-y-2">
                  <div className="h-4 w-32 rounded-md bg-muted animate-pulse" />
                  <div className="h-3 w-48 rounded-md bg-muted/60 animate-pulse" />
                </div>
                <div className="h-[280px] flex items-end gap-2 px-4 pb-4">
                  {[40, 65, 30, 80, 55, 70, 45, 60, 35, 75, 50, 68].map((h, j) => (
                    <div
                      key={j}
                      className="flex-1 rounded-t-md bg-muted/50 animate-pulse"
                      style={{ height: `${h}%`, animationDelay: `${j * 80}ms` }}
                    />
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : !hasData ? (
        <Card>
          <CardContent className="p-6">
            <div className="text-center py-12">
              <p className="text-muted-foreground">{t('usageCharts.emptyTitle')}</p>
              <p className="text-sm text-muted-foreground/70 mt-1">{t('usageCharts.emptyDesc')}</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {/* 请求量趋势 */}
          <ChartCard title={t('usageCharts.requestTrend')} description={t('usageCharts.requestTrendDesc')}>
            <ResponsiveContainer width="100%" height="100%">
              <ComposedChart data={displayData.timelineData} margin={chartMargin}>
                <defs>
                  <linearGradient id="request-gradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.28} />
                    <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid vertical={false} stroke={gridColor} strokeDasharray="4 4" />
                <XAxis
                  dataKey="label"
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  minTickGap={20}
                  tickMargin={8}
                />
                <YAxis
                  tickFormatter={formatCompactNumber}
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  allowDecimals={false}
                  tickCount={8}
                />
                <Tooltip
                  position={{ y: 10 }}
                  formatter={(value) => formatNumber(value)}
                  labelFormatter={(_, payload) => {
                    const p = payload?.[0]?.payload as TimelineDisplayPoint | undefined
                    return p?.fullLabel ?? ''
                  }}
                  contentStyle={tooltipContentStyle}
                  labelStyle={tooltipLabelStyle}
                  itemStyle={tooltipItemStyle}
                />
                <Legend wrapperStyle={{ paddingTop: 4, fontSize: 12 }} />
                <Area
                  type="monotone"
                  dataKey="requests"
                  name={t('usageCharts.seriesRequests')}
                  stroke="var(--color-primary)"
                  fill="url(#request-gradient)"
                  strokeWidth={2.5}
                />
              </ComposedChart>
            </ResponsiveContainer>
          </ChartCard>

          {/* Token 使用趋势 */}
          <ChartCard title={t('usageCharts.tokenTrend')} description={t('usageCharts.tokenTrendDesc')}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={displayData.timelineData} margin={chartMargin}>
                <CartesianGrid vertical={false} stroke={gridColor} strokeDasharray="4 4" />
                <XAxis
                  dataKey="label"
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  minTickGap={20}
                  tickMargin={8}
                />
                <YAxis
                  tickFormatter={formatCompactNumber}
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                />
                <Tooltip
                  position={{ y: 10 }}
                  formatter={(value) => formatNumber(value)}
                  labelFormatter={(_, payload) => {
                    const p = payload?.[0]?.payload as TimelineDisplayPoint | undefined
                    return p?.fullLabel ?? ''
                  }}
                  contentStyle={tooltipContentStyle}
                  labelStyle={tooltipLabelStyle}
                  itemStyle={tooltipItemStyle}
                />
                <Legend wrapperStyle={{ paddingTop: 12, fontSize: 12, color: axisColor }} />
                <Line
                  type="monotone"
                  dataKey="inputTokens"
                  name={t('usageCharts.seriesInputTokens')}
                  stroke="hsl(var(--info))"
                  strokeWidth={2.5}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                <Line
                  type="monotone"
                  dataKey="outputTokens"
                  name={t('usageCharts.seriesOutputTokens')}
                  stroke="hsl(var(--success))"
                  strokeWidth={2.5}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                <Line
                  type="monotone"
                  dataKey="reasoningTokens"
                  name={t('usageCharts.seriesReasoningTokens')}
                  stroke="hsl(36 90% 55%)"
                  strokeWidth={2.5}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </ChartCard>

          {/* 错误率趋势 */}
          <ChartCard title={t('usageCharts.errorRateTrend')} description={t('usageCharts.errorRateTrendDesc')}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={displayData.timelineData} margin={chartMargin}>
                <CartesianGrid vertical={false} stroke={gridColor} strokeDasharray="4 4" />
                <XAxis
                  dataKey="label"
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  minTickGap={20}
                  tickMargin={8}
                />
                <YAxis
                  tickFormatter={(v) => `${v}%`}
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  domain={[0, 'auto']}
                />
                <Tooltip
                  position={{ y: 10 }}
                  formatter={(value) => `${value}%`}
                  labelFormatter={(_, payload) => {
                    const p = payload?.[0]?.payload as TimelineDisplayPoint | undefined
                    return p?.fullLabel ?? ''
                  }}
                  contentStyle={tooltipContentStyle}
                  labelStyle={tooltipLabelStyle}
                  itemStyle={tooltipItemStyle}
                />
                <Legend wrapperStyle={{ paddingTop: 4, fontSize: 12 }} />
                <Line
                  type="monotone"
                  dataKey="errorRate"
                  name={t('usageCharts.seriesErrorRate')}
                  stroke="var(--color-destructive)"
                  strokeWidth={2.5}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </ChartCard>

          {/* 延迟趋势 */}
          <ChartCard title={t('usageCharts.latencyTrend')} description={t('usageCharts.latencyTrendDesc')}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={displayData.timelineData} margin={chartMargin}>
                <CartesianGrid vertical={false} stroke={gridColor} strokeDasharray="4 4" />
                <XAxis
                  dataKey="label"
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  minTickGap={20}
                  tickMargin={8}
                />
                <YAxis
                  tickFormatter={formatDuration}
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  width={54}
                />
                <Tooltip
                  position={{ y: 10 }}
                  formatter={(value) => formatDuration(value)}
                  labelFormatter={(_, payload) => {
                    const p = payload?.[0]?.payload as TimelineDisplayPoint | undefined
                    return p?.fullLabel ?? ''
                  }}
                  contentStyle={tooltipContentStyle}
                  labelStyle={tooltipLabelStyle}
                  itemStyle={tooltipItemStyle}
                />
                <Line
                  type="monotone"
                  dataKey="avgLatency"
                  name={t('usageCharts.seriesAvgLatency')}
                  stroke="hsl(var(--info))"
                  strokeWidth={2.5}
                  dot={false}
                  connectNulls
                  activeDot={{ r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </ChartCard>

          {/* 按模型统计 */}
          <ChartCard title={t('usageCharts.modelRanking')} description={t('usageCharts.modelRankingDesc')}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={displayData.modelData} layout="vertical" margin={{ top: 8, right: 12, left: 8, bottom: 0 }}>
                <CartesianGrid horizontal={false} stroke={gridColor} strokeDasharray="4 4" />
                <XAxis
                  type="number"
                  tickFormatter={formatCompactNumber}
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  allowDecimals={false}
                />
                <YAxis
                  dataKey="shortModel"
                  type="category"
                  width={128}
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                />
                <Tooltip
                  position={{ y: 10 }}
                  formatter={(value) => formatNumber(value)}
                  labelFormatter={(_, payload) => {
                    const p = payload?.[0]?.payload as ModelRankingPoint | undefined
                    return p?.model ?? ''
                  }}
                  contentStyle={tooltipContentStyle}
                  labelStyle={tooltipLabelStyle}
                  itemStyle={tooltipItemStyle}
                />
                <Bar dataKey="requests" name={t('usageCharts.seriesRequests')} fill="hsl(var(--success))" radius={[0, 8, 8, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>

          {/* 按账号统计 */}
          <ChartCard title={t('usageCharts.accountRanking')} description={t('usageCharts.accountRankingDesc')}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={displayData.accountData} layout="vertical" margin={{ top: 8, right: 12, left: 8, bottom: 0 }}>
                <CartesianGrid horizontal={false} stroke={gridColor} strokeDasharray="4 4" />
                <XAxis
                  type="number"
                  tickFormatter={formatCompactNumber}
                  tick={{ fill: axisColor, fontSize: 12 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                  allowDecimals={false}
                />
                <YAxis
                  dataKey="shortAccount"
                  type="category"
                  width={128}
                  tick={{ fill: axisColor, fontSize: 10 }}
                  axisLine={{ stroke: gridColor }}
                  tickLine={{ stroke: gridColor }}
                />
                <Tooltip
                  position={{ y: 10 }}
                  formatter={(value) => formatNumber(value)}
                  labelFormatter={(_, payload) => {
                    const p = payload?.[0]?.payload as AccountRankingPoint | undefined
                    return p?.account ?? ''
                  }}
                  contentStyle={tooltipContentStyle}
                  labelStyle={tooltipLabelStyle}
                  itemStyle={tooltipItemStyle}
                />
                <Bar dataKey="requests" name={t('usageCharts.seriesRequests')} fill="hsl(var(--info))" radius={[0, 8, 8, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>
        </div>
      )}
    </div>
  )
}

function ChartCard({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
  return (
    <Card className="py-0">
      <CardContent className="p-6">
        <div className="mb-5">
          <h4 className="text-base font-semibold text-foreground">{title}</h4>
          <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{description}</p>
        </div>
        <div className="h-[280px]">{children}</div>
      </CardContent>
    </Card>
  )
}
