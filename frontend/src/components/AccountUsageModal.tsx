import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts'
import Modal from './Modal'
import { api } from '../api'
import type { AccountRow, AccountUsageDetail } from '../types'
import { getErrorMessage } from '../utils/error'

const COLORS = ['#7c3aed', '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#ec4899', '#8b5cf6', '#06b6d4', '#84cc16', '#f97316']

interface Props {
  account: AccountRow
  onClose: () => void
}

export default function AccountUsageModal({ account, onClose }: Props) {
  const { t } = useTranslation()
  const [data, setData] = useState<AccountUsageDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [creditEnabled, setCreditEnabled] = useState(account.credit_enabled ?? false)
  const [creditSkipUsageWindow, setCreditSkipUsageWindow] = useState(account.credit_skip_usage_window ?? false)
  const [savingCredit, setSavingCredit] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await api.getAccountUsage(account.id)
      setData(result)
    } catch (err) {
      setError(getErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [account.id])

  useEffect(() => { void load() }, [load])

  const handleCreditToggle = async (field: 'credit_enabled' | 'credit_skip_usage_window') => {
    setSavingCredit(true)
    try {
      const newCreditEnabled = field === 'credit_enabled' ? !creditEnabled : creditEnabled
      const newCreditSkipUsageWindow = field === 'credit_skip_usage_window' ? !creditSkipUsageWindow : creditSkipUsageWindow
      await api.updateAccountCredit(account.id, {
        credit_enabled: newCreditEnabled,
        credit_skip_usage_window: newCreditSkipUsageWindow,
      })
      if (field === 'credit_enabled') {
        setCreditEnabled(newCreditEnabled)
        account.credit_enabled = newCreditEnabled
      } else {
        setCreditSkipUsageWindow(newCreditSkipUsageWindow)
        account.credit_skip_usage_window = newCreditSkipUsageWindow
      }
    } catch (err) {
      // silently fail, the toggle will revert
    } finally {
      setSavingCredit(false)
    }
  }

  const title = t('accounts.usageDetailTitle') + ' — ' + (account.email || account.name || `#${account.id}`)

  return (
    <Modal show title={title} onClose={onClose} contentClassName="sm:max-w-[720px]">
      {loading ? (
        <div className="flex items-center justify-center py-12 text-muted-foreground text-sm">{t('common.loading')}</div>
      ) : error ? (
        <div className="py-8 text-center text-sm text-red-500">{error}</div>
      ) : !data || data.total_requests === 0 ? (
        <div className="py-12 text-center text-sm text-muted-foreground">{t('accounts.noUsageData')}</div>
      ) : (
        <div className="flex gap-6">
          {/* 左侧：饼图 */}
          <div className="shrink-0">
            <h4 className="text-sm font-semibold mb-2">{t('accounts.modelDistribution')}</h4>
            <div className="w-[200px] h-[200px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={data.models}
                    dataKey="requests"
                    nameKey="model"
                    cx="50%"
                    cy="50%"
                    innerRadius={45}
                    outerRadius={85}
                    paddingAngle={2}
                    strokeWidth={0}
                  >
                    {data.models.map((_, i) => (
                      <Cell key={i} fill={COLORS[i % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value: number, name: string) => [`${value} 次`, name]}
                    contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid hsl(var(--border))' }}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
            {/* 图例 */}
            <div className="mt-2 space-y-1">
              {data.models.map((m, i) => (
                <div key={m.model} className="flex items-center gap-2 text-[12px]">
                  <span className="size-2.5 rounded-full shrink-0" style={{ background: COLORS[i % COLORS.length] }} />
                  <span className="truncate text-foreground font-medium">{m.model}</span>
                  <span className="ml-auto shrink-0 text-muted-foreground tabular-nums">{m.requests.toLocaleString()}</span>
                </div>
              ))}
            </div>
          </div>

          {/* 右侧：Token 统计 */}
          <div className="flex-1 space-y-2.5">
            <StatRow label={t('accounts.totalRequests')} value={data.total_requests.toLocaleString()} highlight />
            <StatRow label={t('accounts.totalTokens')} value={data.total_tokens.toLocaleString()} highlight />
            <div className="h-px bg-border" />
            <StatRow label={t('accounts.inputTokens')} value={data.input_tokens.toLocaleString()} />
            <StatRow label={t('accounts.outputTokens')} value={data.output_tokens.toLocaleString()} />
            <StatRow label={t('accounts.reasoningTokens')} value={data.reasoning_tokens.toLocaleString()} />
            <StatRow label={t('accounts.cachedTokens')} value={data.cached_tokens.toLocaleString()} />
          </div>
        </div>
      )}

      {/* Credit 设置 */}
      <div className="mt-4 pt-4 border-t border-border">
        <h4 className="text-sm font-semibold mb-3">{t('accounts.creditBadge')}</h4>
        <div className="space-y-3">
          <label className="flex items-center justify-between gap-3 rounded-lg border border-border px-3.5 py-2.5 cursor-pointer hover:bg-muted/30 transition-colors">
            <div className="min-w-0">
              <div className="text-[13px] font-medium text-foreground">{t('accounts.creditEnabled')}</div>
              <div className="text-[11px] text-muted-foreground mt-0.5">{t('settings.creditEnabledDesc')}</div>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={creditEnabled}
              disabled={savingCredit}
              onClick={() => void handleCreditToggle('credit_enabled')}
              className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
                creditEnabled ? 'bg-violet-500' : 'bg-muted'
              }`}
            >
              <span
                className="inline-block size-3.5 rounded-full bg-white shadow-sm transition-transform"
                style={{ transform: creditEnabled ? 'translateX(18px)' : 'translateX(3px)' }}
              />
            </button>
          </label>
          <label className="flex items-center justify-between gap-3 rounded-lg border border-border px-3.5 py-2.5 cursor-pointer hover:bg-muted/30 transition-colors">
            <div className="min-w-0">
              <div className="text-[13px] font-medium text-foreground">{t('accounts.creditSkipUsageWindow')}</div>
              <div className="text-[11px] text-muted-foreground mt-0.5">{t('settings.creditSkipUsageWindowDesc')}</div>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={creditSkipUsageWindow}
              disabled={savingCredit}
              onClick={() => void handleCreditToggle('credit_skip_usage_window')}
              className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
                creditSkipUsageWindow ? 'bg-violet-500' : 'bg-muted'
              }`}
            >
              <span
                className="inline-block size-3.5 rounded-full bg-white shadow-sm transition-transform"
                style={{ transform: creditSkipUsageWindow ? 'translateX(18px)' : 'translateX(3px)' }}
              />
            </button>
          </label>
        </div>
      </div>
    </Modal>
  )
}

function StatRow({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="flex items-center justify-between rounded-lg border border-border px-3.5 py-2">
      <span className="text-[13px] text-muted-foreground">{label}</span>
      <span className={`tabular-nums font-semibold ${highlight ? 'text-[15px] text-foreground' : 'text-[14px] text-foreground/80'}`}>{value}</span>
    </div>
  )
}
