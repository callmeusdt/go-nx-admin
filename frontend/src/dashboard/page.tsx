import React, { useContext, useMemo } from 'react'
import { useI18n } from '../contexts/i18n-context'
import { builtinWidgets, StatsContext } from './builtin'
import { DashboardOptionsContext } from './context'
import { resolveWidgets, type DashboardWidget } from './registry'
import { useDashboardData } from './widgets'
import { useMyPermissions } from './permissions'

const SPAN: Record<number, string> = {
  1: 'lg:col-span-1',
  2: 'lg:col-span-2',
  3: 'lg:col-span-3',
  4: 'lg:col-span-4',
}

export const DashboardPage: React.FC = () => {
  const { t } = useI18n()
  const options = useContext(DashboardOptionsContext)
  const stats = useDashboardData<Record<string, number>>('/dashboard/stats')
  const granted = useMyPermissions()

  const widgets = useMemo(() => {
    const all = resolveWidgets(builtinWidgets, options)
    return all.filter(widget => !widget.permission || granted === 'all' || granted.has(widget.permission))
  }, [options, granted])

  const Custom = options?.page
  if (Custom) return <Custom />

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-slate-800">{t('dashboard.title')}</h2>
      <StatsContext.Provider value={{ stats: stats.data, loading: stats.loading, failed: stats.failed }}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {widgets.map((widget: DashboardWidget) => {
            const Widget = widget.component
            return (
              <div key={widget.key} data-widget={widget.key} className={`min-w-0 ${SPAN[widget.span ?? 1]} ${(widget.span ?? 1) > 1 ? 'md:col-span-2' : ''}`}>
                <Widget />
              </div>
            )
          })}
        </div>
      </StatsContext.Provider>
    </div>
  )
}
