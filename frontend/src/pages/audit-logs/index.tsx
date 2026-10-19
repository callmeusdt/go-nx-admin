import React, { useEffect, useState, useCallback } from 'react'
import { api } from '../../lib/api'
import { AuditLog } from '../../types'
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

const methodVariants: Record<string, 'info' | 'success' | 'warning' | 'danger'> = {
  GET: 'info',
  POST: 'success',
  PUT: 'warning',
  DELETE: 'danger',
}

export const AuditLogsPage: React.FC = () => {
  const { t, locale } = useI18n()
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [loadFailed, setLoadFailed] = useState(false)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState({ operator: '', path: '', method: '', ip: '', start: '', end: '' })

  const fetchLogs = useCallback(async (nextPage = page, appliedFilters = filters) => {
    setLoadFailed(false)
    try {
      const params = new URLSearchParams({ page: String(nextPage), page_size: String(pageSize) })
      Object.entries(appliedFilters).forEach(([key, value]) => value && params.set(key, value))
      const res = await api.get<{ data: AuditLog[]; total: number }>(`/audit-logs?${params.toString()}`)
      setLogs(res.data)
      setTotal(res.total)
      setPage(nextPage)
    } catch {
      setLoadFailed(true)
    } finally {
      setLoading(false)
    }
  }, [filters, page])

  useEffect(() => { fetchLogs(1) }, [])
  useOnTabActivate(() => { void fetchLogs(page) })


  return (
    <div className="space-y-4">
      <PageHeader title={t('audit_logs.title')} />
      {loadFailed && <Alert>{t('ui.request_failed')}</Alert>}

      <FilterBar
        onQuery={() => fetchLogs(1)}
        onReset={() => { const reset = { operator: '', path: '', method: '', ip: '', start: '', end: '' }; setFilters(reset); void fetchLogs(1, reset) }}
      >
        <FilterField><Input placeholder={t('audit_logs.operator')} aria-label={t('audit_logs.operator')} value={filters.operator} onChange={e => setFilters(f => ({ ...f, operator: e.target.value }))} /></FilterField>
        <FilterField><Input placeholder={t('audit_logs.path')} aria-label={t('audit_logs.path')} value={filters.path} onChange={e => setFilters(f => ({ ...f, path: e.target.value }))} /></FilterField>
        <FilterField>
          <Select
            value={filters.method}
            onValueChange={method => setFilters(f => ({ ...f, method }))}
            aria-label={t('logs.method')}
            options={[{ value: '', label: t('audit_logs.all_methods') }, ...['GET', 'POST', 'PUT', 'DELETE'].map(m => ({ value: m, label: m }))]}
          />
        </FilterField>
        <FilterField><Input placeholder="IP" aria-label="IP" value={filters.ip} onChange={e => setFilters(f => ({ ...f, ip: e.target.value }))} /></FilterField>
        <FilterField><DateTimeInput value={filters.start} placeholder={t('logs.start')} onChange={value => setFilters(f => ({ ...f, start: value }))} /></FilterField>
        <FilterField><DateTimeInput value={filters.end} placeholder={t('logs.end')} onChange={value => setFilters(f => ({ ...f, end: value }))} /></FilterField>
      </FilterBar>

      <DataTable<AuditLog>
        storageKey="audit-logs"
        rows={logs}
        rowKey={log => log.id}
        loading={loading}
        page={page}
        pageSize={pageSize}
        total={total}
        onPageChange={fetchLogs}
        emptyText={t('audit_logs.empty')}
        columns={[
          { key: 'id', title: 'ID', className: 'text-gray-500' },
          { key: 'operator', title: t('audit_logs.operator'), label: t('audit_logs.operator'), className: 'font-medium' },
          { key: 'path', title: t('audit_logs.path'), label: t('audit_logs.path'), className: 'text-xs font-mono text-gray-600' },
          {
            key: 'method', title: t('logs.method'), label: t('logs.method'),
            render: log => (
              <Badge variant={methodVariants[log.method] || 'neutral'} className="font-mono">{log.method}</Badge>
            ),
          },
          { key: 'ip', title: 'IP', className: 'text-gray-500 text-xs' },
          { key: 'duration', title: t('logs.duration'), label: t('logs.duration'), className: 'text-gray-500 text-xs', render: log => `${log.duration}μs` },
          { key: 'created_at', title: t('logs.time'), label: t('logs.time'), className: 'text-gray-500 text-xs', render: log => new Date(log.created_at).toLocaleString(locale) },
        ]}
      />
    </div>
  )
}
