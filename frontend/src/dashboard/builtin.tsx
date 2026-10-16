import React from 'react'
import { Activity, LayoutDashboard, Shield, Users } from 'lucide-react'
import { useI18n } from '../contexts/i18n-context'
import type { DashboardWidget } from './registry'
import { StatCard, WidgetState } from './widgets'

interface Stats { total_users?: number; online_users?: number; total_roles?: number; today_logins?: number }

/** The page fetches /dashboard/stats once and every built-in card reads it from here. */
export const StatsContext = React.createContext<{ stats: Stats | null; loading: boolean; failed: boolean }>({ stats: null, loading: true, failed: false })

function builtinCard(
  field: keyof Stats,
  labelKey: string,
  icon: React.ComponentType<{ className?: string }>,
  color: string,
): React.FC {
  const Card: React.FC = () => {
    const { t } = useI18n()
    const { stats, loading, failed } = React.useContext(StatsContext)
    if (!stats) return <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100 h-full"><WidgetState loading={loading} failed={failed} /></div>
    return <StatCard label={t(labelKey)} value={String(stats[field] ?? 0)} icon={icon} color={color} />
  }
  Card.displayName = `Dashboard.${String(field)}`
  return Card
}

export const builtinWidgets: DashboardWidget[] = [
  { key: 'builtin.total-users', order: 10, component: builtinCard('total_users', 'dashboard.users', Users, 'bg-blue-500') },
  { key: 'builtin.online-users', order: 20, component: builtinCard('online_users', 'dashboard.online', Activity, 'bg-green-500') },
  { key: 'builtin.total-roles', order: 30, component: builtinCard('total_roles', 'dashboard.roles', Shield, 'bg-purple-500') },
  { key: 'builtin.today-logins', order: 40, component: builtinCard('today_logins', 'dashboard.logins', LayoutDashboard, 'bg-orange-500') },
]
