import React, { useEffect, useState, useCallback } from 'react'
import { Loader2 } from 'lucide-react'
import { api } from '../../lib/api'
import { AuditLog } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { DateTimeInput } from '../../components/ui/datetime-input'

const pageSize = 20

const methodColors: Record<string, string> = {
  'GET': 'bg-blue-50 text-blue-700',
  'POST': 'bg-green-50 text-green-700',
  'PUT': 'bg-amber-50 text-amber-700',
  'DELETE': 'bg-red-50 text-red-700',
}

export const AuditLogsPage: React.FC = () => {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState({ operator: '', path: '', method: '', ip: '', start: '', end: '' })

  const fetchLogs = useCallback(async (nextPage = page) => {
    try {
      const params = new URLSearchParams({ page: String(nextPage), page_size: String(pageSize) })
      Object.entries(filters).forEach(([key, value]) => value && params.set(key, value))
      const res = await api.get<{ data: AuditLog[]; total: number }>(`/audit-logs?${params.toString()}`)
      setLogs(res.data)
      setTotal(res.total)
      setPage(nextPage)
    } finally {
      setLoading(false)
    }
  }, [filters, page])

  useEffect(() => { fetchLogs(1) }, [])

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-6 h-6 animate-spin text-gray-400" /></div>
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-slate-800">操作日志</h2>
        <div className="flex items-center gap-2" />
      </div>

      <div className="grid grid-cols-6 gap-2 rounded-lg border border-gray-200 bg-white p-3">
        <Input placeholder="操作人" value={filters.operator} onChange={e => setFilters(f => ({ ...f, operator: e.target.value }))} />
        <Input placeholder="请求路径" value={filters.path} onChange={e => setFilters(f => ({ ...f, path: e.target.value }))} />
        <select className="h-10 rounded-md border border-gray-300 px-3 text-sm" value={filters.method} onChange={e => setFilters(f => ({ ...f, method: e.target.value }))}>
          <option value="">全部方法</option><option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option>
        </select>
        <Input placeholder="IP" value={filters.ip} onChange={e => setFilters(f => ({ ...f, ip: e.target.value }))} />
        <DateTimeInput value={filters.start} placeholder="开始时间" onChange={value => setFilters(f => ({ ...f, start: value }))} />
        <DateTimeInput value={filters.end} placeholder="结束时间" onChange={value => setFilters(f => ({ ...f, end: value }))} />
        <div className="col-span-6 flex justify-end gap-2">
          <Button variant="outline" onClick={() => { setFilters({ operator: '', path: '', method: '', ip: '', start: '', end: '' }); setTimeout(() => fetchLogs(1), 0) }}>重置</Button>
          <Button onClick={() => fetchLogs(1)}>查询</Button>
        </div>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-3 font-medium text-gray-600">ID</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">操作人</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">请求路径</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">方法</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">IP</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">耗时</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">时间</th>
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
                <td className="px-4 py-2.5 text-gray-500 text-xs">{new Date(log.created_at).toLocaleString()}</td>
              </tr>
            ))}
            {logs.length === 0 && (
              <tr><td colSpan={7} className="text-center py-8 text-gray-400">暂无操作记录</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="flex items-center justify-between text-sm text-gray-500">
        <span>共 {total} 条</span>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => fetchLogs(page - 1)}>上一页</Button>
          <span>第 {page} / {totalPages} 页</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => fetchLogs(page + 1)}>下一页</Button>
        </div>
      </div>
    </div>
  )
}
