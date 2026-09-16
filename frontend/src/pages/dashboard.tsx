import React, { useEffect, useState } from 'react'
import { LayoutDashboard, Users, Shield, Activity } from 'lucide-react'
import { api } from '../lib/api'
import { DashboardStats } from '../types'
import { useI18n } from '../contexts/i18n-context'

const iconMap: Record<string, React.FC<{ className?: string }>> = {
  total_users: Users,
  online_users: Activity,
  total_roles: Shield,
  today_logins: LayoutDashboard,
}

const colorMap: Record<string, string> = {
  total_users: 'bg-blue-500',
  online_users: 'bg-green-500',
  total_roles: 'bg-purple-500',
  today_logins: 'bg-orange-500',
}

const labelMap: Record<string, string> = {
  total_users: 'dashboard.users',
  online_users: 'dashboard.online',
  total_roles: 'dashboard.roles',
  today_logins: 'dashboard.logins',
}

export const DashboardPage: React.FC = () => {
  const { t } = useI18n()
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    api.get<DashboardStats>('/dashboard/stats').then(setStats).catch(() => setFailed(true))
  }, [])

  if (!stats) {
    if (failed) return <p role="alert" className="text-red-600 py-12">{t('ui.request_failed')}</p>
    return <div className="text-center text-slate-400 py-12">{t('common.loading')}</div>
  }

  const items = Object.entries(stats).filter(([k]) => k !== 'today_api_calls')

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-slate-800">{t('dashboard.title')}</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {items.map(([key, value]) => {
          const Icon = iconMap[key] || Users
          const color = colorMap[key] || 'bg-slate-500'
          const label = labelMap[key] || key
          return (
            <div key={key} className="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">{t(label)}</p>
                  <p className="text-2xl font-bold text-slate-800 mt-1">{String(value)}</p>
                </div>
                <div className={`w-12 h-12 ${color} rounded-lg flex items-center justify-center`}>
                  <Icon className="w-6 h-6 text-white" />
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
