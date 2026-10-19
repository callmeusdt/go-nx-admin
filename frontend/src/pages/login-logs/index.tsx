import React, { useEffect, useState, useCallback } from 'react'
import { api } from '../../lib/api'
import { LoginLog } from '../../types'
import { Alert } from '../../components/ui/alert'
import { Badge } from '../../components/ui/badge'
import { PageHeader } from '../../components/ui/page-header'
import { FilterBar, FilterField } from '../../components/ui/filter-bar'
import { Input } from '../../components/ui/input'
import { Select } from '../../components/ui/select'
import { DateTimeInput } from '../../components/ui/datetime-input'
import { DataTable } from '../../components/ui/data-table'
import { useI18n } from '../../contexts/i18n-context'
import { useOnTabActivate } from '../../components/layout/keep-alive'

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
  useOnTabActivate(() => { void load(page) })

  return (
    <div className="space-y-4">
      <PageHeader title={t('login_logs.title')} />
      {loadFailed && <Alert>{t('ui.request_failed')}</Alert>}
      <FilterBar
        onQuery={() => load(1)}
        onReset={() => { setFilters(defaultFilters); void load(1, defaultFilters) }}
      >
        <FilterField><Input placeholder={t('login_logs.username')} aria-label={t('login_logs.username')} value={filters.username} onChange={e => setFilters(f => ({ ...f, username: e.target.value }))} /></FilterField>
        <FilterField>
          <Select
            value={filters.status}
            onValueChange={status => setFilters(f => ({ ...f, status }))}
            aria-label={t('login_logs.status')}
            options={[
              { value: '', label: t('login_logs.all_status') },
              { value: 'success', label: t('login_logs.success') },
              { value: 'fail', label: t('login_logs.fail') },
            ]}
          />
        </FilterField>
        <FilterField><Input placeholder="IP" aria-label="IP" value={filters.ip} onChange={e => setFilters(f => ({ ...f, ip: e.target.value }))} /></FilterField>
        <FilterField><DateTimeInput value={filters.start} placeholder={t('logs.start')} onChange={value => setFilters(f => ({ ...f, start: value }))} /></FilterField>
        <FilterField><DateTimeInput value={filters.end} placeholder={t('logs.end')} onChange={value => setFilters(f => ({ ...f, end: value }))} /></FilterField>
      </FilterBar>
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
            render: log => <Badge variant={log.status === 'success' ? 'success' : 'danger'}>{translated('login_logs.' + log.status, log.status)}</Badge>,
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
