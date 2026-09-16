import React, { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { api } from '../../lib/api'
import { AuditLog } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { DateTimeInput } from '../../components/ui/datetime-input'
import { useI18n } from '../../contexts/i18n-context'

const pageSize = 20

const methodColors: Record<string, string> = {
  'GET': 'bg-blue-50 text-blue-700',
  'POST': 'bg-green-50 text-green-700',
  'PUT': 'bg-amber-50 text-amber-700',
  'DELETE': 'bg-red-50 text-red-700',
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

  if (loading) return <div className="flex justify-center py-20" role="status" aria-label={t('common.loading')}><Loader2 className="w-6 h-6 animate-spin text-gray-400" /></div>
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-slate-800">{t('audit_logs.title')}</h2>
        <div className="flex items-center gap-2" />
      </div>
      {loadFailed && <p role="alert" className="text-red-600">{t('ui.request_failed')}</p>}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-2 rounded-lg border border-gray-200 bg-white p-3">
        <Input placeholder={t('audit_logs.operator')} value={filters.operator} onChange={e => setFilters(f => ({ ...f, operator: e.target.value }))} />
        <Input placeholder={t('audit_logs.path')} value={filters.path} onChange={e => setFilters(f => ({ ...f, path: e.target.value }))} />
        <select className="h-10 rounded-md border border-gray-300 px-3 text-sm" value={filters.method} onChange={e => setFilters(f => ({ ...f, method: e.target.value }))}>
          <option value="">{t('audit_logs.all_methods')}</option><option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option>
        </select>
        <Input placeholder="IP" value={filters.ip} onChange={e => setFilters(f => ({ ...f, ip: e.target.value }))} />
        <DateTimeInput value={filters.start} placeholder={t('logs.start')} onChange={value => setFilters(f => ({ ...f, start: value }))} />
        <DateTimeInput value={filters.end} placeholder={t('logs.end')} onChange={value => setFilters(f => ({ ...f, end: value }))} />
        <div className="col-span-full flex justify-end gap-2">
          <Button variant="outline" onClick={() => { const reset = { operator: '', path: '', method: '', ip: '', start: '', end: '' }; setFilters(reset); void fetchLogs(1, reset) }}>{t('logs.reset')}</Button>
          <Button onClick={() => fetchLogs(1)}>{t('logs.query')}</Button>
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-x-auto">
        <table className="w-full min-w-[720px] text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-3 font-medium text-gray-600">ID</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('audit_logs.operator')}</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('audit_logs.path')}</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('logs.method')}</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">IP</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('logs.duration')}</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('logs.time')}</th>
            </tr>
          </thead>
          <tbody>
            {logs.map(log => (
              <tr key={log.id} className="border-b border-gray-100 hover:bg-gray-50">
                <td className="px-4 py-2.5 text-gray-500">{log.id}</td>
                <td className="px-4 py-2.5 font-medium">{log.operator}</td>
                <td className="px-4 py-2.5 text-xs font-mono text-gray-600">{log.path}</td>
                <td className="px-4 py-2.5">
                  <span className={`inline-flex px-1.5 py-0.5 rounded text-xs font-mono ${methodColors[log.method] || 'bg-gray-50 text-gray-600'}`}>
                    {log.method}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-gray-500 text-xs">{log.ip}</td>
                <td className="px-4 py-2.5 text-gray-500 text-xs">{log.duration}μs</td>
                <td className="px-4 py-2.5 text-gray-500 text-xs">{new Date(log.created_at).toLocaleString(locale)}</td>
              </tr>
            ))}
            {logs.length === 0 && (
              <tr><td colSpan={7} className="text-left px-4 py-8 text-gray-400">{t('audit_logs.empty')}</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap gap-2 items-center justify-between text-sm text-gray-500">
        <span>{t('logs.total').replace('{total}', String(total))}</span>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => fetchLogs(page - 1)}>{t('logs.previous')}</Button>
          <span>{t('logs.page').replace('{page}', String(page)).replace('{pages}', String(totalPages))}</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => fetchLogs(page + 1)}>{t('logs.next')}</Button>
        </div>
      </div>
    </div>
  )
}
