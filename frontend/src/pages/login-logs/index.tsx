import React, { useEffect, useState, useCallback } from 'react'
import { api } from '../../lib/api'
import { LoginLog } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { DateTimeInput } from '../../components/ui/datetime-input'
import { DataTable } from '../../components/ui/data-table'
import { useI18n } from '../../contexts/i18n-context'

const pageSize = 20

const pad = (value: number) => String(value).padStart(2, '0')
const dateValue = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
const defaultStartDate = () => {
  const date = new Date()
  date.setDate(date.getDate() - 7)
  return `${dateValue(date)} 00:00`
}
const buildDateTime = (value: string) => value ? `${value}:00` : ''

export const LoginLogsPage: React.FC = () => {
  const { t, locale } = useI18n()
  const translated = (key: string, fallback: string) => { const value = t(key); return value === key ? fallback : value }
  const [logs, setLogs] = useState<LoginLog[]>([])
  const [loading, setLoading] = useState(true)
  const [loadFailed, setLoadFailed] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const defaultFilters = { username: '', status: '', ip: '', start: defaultStartDate(), end: '' }
  const [filters, setFilters] = useState(defaultFilters)

  const load = useCallback(async (nextPage = page, appliedFilters = filters) => {
    setLoadFailed(false)
    try {
      const params = new URLSearchParams({ page: String(nextPage), page_size: String(pageSize) })
      if (appliedFilters.username) params.set('username', appliedFilters.username)
      if (appliedFilters.status) params.set('status', appliedFilters.status)
      if (appliedFilters.ip) params.set('ip', appliedFilters.ip)
      const start = buildDateTime(appliedFilters.start)
      const end = buildDateTime(appliedFilters.end)
      if (start) params.set('start', start)
      if (end) params.set('end', end)
      const res = await api.get<{ data: LoginLog[]; total: number }>(`/login-logs?${params.toString()}`)
      setLogs(res.data)
      setTotal(res.total)
      setPage(nextPage)
    } catch {
      setLoadFailed(true)
    } finally {
      setLoading(false)
    }
  }, [filters, page])

  useEffect(() => { load(1) }, [])

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-slate-800">{t('login_logs.title')}</h2>
        <div className="flex items-center gap-2" />
      </div>
      {loadFailed && <p role="alert" className="text-red-600">{t('ui.request_failed')}</p>}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-2 rounded-lg border border-gray-200 bg-white p-3">
        <Input placeholder={t('login_logs.username')} value={filters.username} onChange={e => setFilters(f => ({ ...f, username: e.target.value }))} />
        <select className="h-10 rounded-md border border-gray-300 px-3 text-sm" value={filters.status} onChange={e => setFilters(f => ({ ...f, status: e.target.value }))}>
          <option value="">{t('login_logs.all_status')}</option><option value="success">{t('login_logs.success')}</option><option value="fail">{t('login_logs.fail')}</option>
        </select>
        <Input placeholder="IP" value={filters.ip} onChange={e => setFilters(f => ({ ...f, ip: e.target.value }))} />
        <DateTimeInput value={filters.start} placeholder={t('logs.start')} onChange={value => setFilters(f => ({ ...f, start: value }))} />
        <DateTimeInput value={filters.end} placeholder={t('logs.end')} onChange={value => setFilters(f => ({ ...f, end: value }))} />
        <div className="col-span-full flex justify-end gap-2">
          <Button variant="outline" onClick={() => { setFilters(defaultFilters); void load(1, defaultFilters) }}>{t('logs.reset')}</Button>
          <Button onClick={() => load(1)}>{t('logs.query')}</Button>
        </div>
      </div>
      <DataTable<LoginLog>
        storageKey="login-logs"
        rows={logs}
        rowKey={log => log.id}
        loading={loading}
        page={page}
        pageSize={pageSize}
        total={total}
        onPageChange={load}
        emptyText={loadFailed ? '' : t('common.no_data')}
        columns={[
          { key: 'username', title: t('login_logs.user'), label: t('login_logs.user') },
          {
            key: 'status', title: t('login_logs.status'), label: t('login_logs.status'),
            render: log => <span className={log.status === 'success' ? 'text-green-600' : 'text-red-600'}>{translated('login_logs.' + log.status, log.status)}</span>,
          },
          {
            key: 'ip', title: t('login_logs.ip_geo'), label: t('login_logs.ip_geo'),
            render: log => <><div className="font-mono text-xs">{log.ip || '-'}</div><div className="mt-0.5 text-xs text-gray-500">{log.geo || '-'}</div></>,
          },
          { key: 'message', title: t('login_logs.message'), label: t('login_logs.message'), render: log => translated('login_logs.reason.' + log.message, log.message) },
          { key: 'created_at', title: t('logs.time'), label: t('logs.time'), className: 'text-xs text-gray-500', render: log => new Date(log.created_at).toLocaleString(locale) },
        ]}
      />
    </div>
  )
}
