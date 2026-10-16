import React, { useCallback, useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useI18n } from '../contexts/i18n-context'
import { useOnTabActivate } from '../components/layout/keep-alive'

interface StatCardProps {
  label: React.ReactNode
  value: React.ReactNode
  icon?: React.ComponentType<{ className?: string }>
  /** Tailwind background class for the icon tile. */
  color?: string
}

/** A single-number card in the stock style. */
export const StatCard: React.FC<StatCardProps> = ({ label, value, icon: Icon, color = 'bg-slate-500' }) => (
  <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100 h-full">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm text-gray-500">{label}</p>
        <p className="text-2xl font-bold text-slate-800 mt-1">{value}</p>
      </div>
      {Icon && (
        <div className={`w-12 h-12 ${color} rounded-lg flex items-center justify-center`}>
          <Icon className="w-6 h-6 text-white" />
        </div>
      )}
    </div>
  </div>
)

/** A titled container for charts, tables or any custom content. */
export const DashboardCard: React.FC<{ title?: React.ReactNode; children: React.ReactNode }> = ({ title, children }) => (
  <div className="bg-white rounded-xl p-6 shadow-sm border border-gray-100 h-full">
    {title && <h3 className="text-sm font-semibold text-slate-700 mb-3">{title}</h3>}
    {children}
  </div>
)

export interface DashboardData<T> {
  data: T | null
  loading: boolean
  failed: boolean
  reload: () => void
}

/**
 * Fetches `path` under /api/v1 and refetches whenever the dashboard tab becomes
 * visible again, so a kept-alive page never shows stale figures.
 */
export function useDashboardData<T = any>(path: string): DashboardData<T> {
  const [data, setData] = useState<T | null>(null)
  const [failed, setFailed] = useState(false)
  const [loading, setLoading] = useState(true)
  const reload = useCallback(() => {
    api.get<T>(path)
      .then(result => { setData(result); setFailed(false) })
      .catch(() => setFailed(true))
      .finally(() => setLoading(false))
  }, [path])
  useEffect(() => { reload() }, [reload])
  useOnTabActivate(reload)
  return { data, loading, failed, reload }
}

/** Standard loading / error placeholder for a widget. */
export const WidgetState: React.FC<{ loading: boolean; failed: boolean }> = ({ loading, failed }) => {
  const { t } = useI18n()
  if (failed) return <p role="alert" className="text-sm text-red-600">{t('ui.request_failed')}</p>
  if (loading) return <p className="text-sm text-slate-400">{t('common.loading')}</p>
  return null
}
