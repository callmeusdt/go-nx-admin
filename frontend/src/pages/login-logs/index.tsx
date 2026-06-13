import React, { useEffect, useState, useCallback } from 'react'
import { api } from '../../lib/api'
import { LoginLog } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { DateTimeInput } from '../../components/ui/datetime-input'

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
  const [logs, setLogs] = useState<LoginLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const defaultFilters = { username: '', status: '', ip: '', start: `${defaultStartDate()} 00:00`, end: '' }
  const [filters, setFilters] = useState(defaultFilters)

  const load = useCallback(async (nextPage = page) => {
    const params = new URLSearchParams({ page: String(nextPage), page_size: String(pageSize) })
    if (filters.username) params.set('username', filters.username)
    if (filters.status) params.set('status', filters.status)
    if (filters.ip) params.set('ip', filters.ip)
    const start = buildDateTime(filters.start)
    const end = buildDateTime(filters.end)
    if (start) params.set('start', start)
    if (end) params.set('end', end)
    const res = await api.get<{ data: LoginLog[]; total: number }>(`/login-logs?${params.toString()}`)
    setLogs(res.data)
    setTotal(res.total)
    setPage(nextPage)
  }, [filters, page])

  useEffect(() => { load(1) }, [])
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-slate-800">登录日志</h2>
        <div className="flex items-center gap-2" />
      </div>
      <div className="grid grid-cols-5 gap-2 rounded-lg border border-gray-200 bg-white p-3">
        <Input placeholder="用户名" value={filters.username} onChange={e => setFilters(f => ({ ...f, username: e.target.value }))} />
        <select className="h-10 rounded-md border border-gray-300 px-3 text-sm" value={filters.status} onChange={e => setFilters(f => ({ ...f, status: e.target.value }))}>
          <option value="">全部状态</option><option value="success">success</option><option value="fail">fail</option>
        </select>
        <Input placeholder="IP" value={filters.ip} onChange={e => setFilters(f => ({ ...f, ip: e.target.value }))} />
        <DateTimeInput value={filters.start} placeholder="开始时间" onChange={value => setFilters(f => ({ ...f, start: value }))} />
        <DateTimeInput value={filters.end} placeholder="结束时间" onChange={value => setFilters(f => ({ ...f, end: value }))} />
        <div className="col-span-5 flex justify-end gap-2">
          <Button variant="outline" onClick={() => { setFilters(defaultFilters); setTimeout(() => load(1), 0) }}>重置</Button>
          <Button onClick={() => load(1)}>查询</Button>
        </div>
      </div>
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr><th className="text-left px-4 py-3">用户</th><th className="text-left px-4 py-3">状态</th><th className="text-left px-4 py-3">IP / 地理位置</th><th className="text-left px-4 py-3">消息</th><th className="text-left px-4 py-3">时间</th></tr></thead><tbody>
          {logs.map(log => <tr key={log.id} className="border-t"><td className="px-4 py-2.5">{log.username}</td><td className="px-4 py-2.5"><span className={log.status === 'success' ? 'text-green-600' : 'text-red-600'}>{log.status}</span></td><td className="px-4 py-2.5"><div className="font-mono text-xs">{log.ip || '-'}</div><div className="mt-0.5 text-xs text-gray-500">{log.geo || '-'}</div></td><td className="px-4 py-2.5">{log.message}</td><td className="px-4 py-2.5 text-xs text-gray-500">{new Date(log.created_at).toLocaleString()}</td></tr>)}
        </tbody></table>
      </div>
      <div className="flex items-center justify-between text-sm text-gray-500">
        <span>共 {total} 条</span>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => load(page - 1)}>上一页</Button>
          <span>第 {page} / {totalPages} 页</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => load(page + 1)}>下一页</Button>
        </div>
      </div>
    </div>
  )
}
