import { NavLink } from 'react-router-dom'
import { Activity, AlertCircle, Workflow } from 'lucide-react'
import { useTranslation } from 'react-i18next'

const tabs = [
  { to: '/ops/overview', labelKey: 'ops.tabs.overview', icon: <Activity className="size-4" /> },
  { to: '/ops/errors', labelKey: 'ops.tabs.errors', icon: <AlertCircle className="size-4" /> },
  { to: '/ops/scheduler', labelKey: 'ops.tabs.scheduler', icon: <Workflow className="size-4" /> },
]

export default function OpsTabs({ compact = false }: { compact?: boolean }) {
  const { t } = useTranslation()

  return (
    <div className={compact ? 'flex flex-wrap items-center gap-1 rounded-lg border border-border bg-muted/30 p-1' : 'flex flex-wrap items-center gap-1 rounded-lg border border-border bg-muted/30 p-1'}>
      {tabs.map((tab) => (
        <NavLink
          key={tab.to}
          to={tab.to}
          className={({ isActive }) =>
            `inline-flex items-center gap-2 rounded-md border text-[13px] font-semibold transition-colors ${compact ? 'h-8 px-2.5' : 'h-9 px-3'} ${
              isActive
                ? 'border-primary/25 bg-primary/10 text-primary'
                : 'border-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground'
            }`
          }
        >
          {tab.icon}
          {t(tab.labelKey)}
        </NavLink>
      ))}
    </div>
  )
}
