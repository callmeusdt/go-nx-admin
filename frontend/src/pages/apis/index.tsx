import React, { useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { ApiItem } from '../../types'
import { Button } from '../../components/ui/button'

export const ApisPage: React.FC = () => {
  const [items, setItems] = useState<ApiItem[]>([])
  const [form, setForm] = useState({ group: '默认', name: '', path: '/api/v1/', method: 'GET', description: '' })
  const load = () => api.get<{ data: ApiItem[] }>('/apis').then(res => setItems(res.data))
  useEffect(() => { load() }, [])

  const create = async () => {
    if (!form.name || !form.path) return
    await api.post('/apis', form)
    setForm({ group: '默认', name: '', path: '/api/v1/', method: 'GET', description: '' })
    load()
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold text-slate-800">API管理</h2>
      <div className="bg-white rounded-lg border border-gray-200 p-4 grid grid-cols-[120px_160px_1fr_110px_1fr_auto] gap-2">
        <input className="border rounded px-3 text-sm" value={form.group} onChange={e => setForm(f => ({ ...f, group: e.target.value }))} placeholder="分组" />
        <input className="border rounded px-3 text-sm" value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="名称" />
        <input className="border rounded px-3 text-sm" value={form.path} onChange={e => setForm(f => ({ ...f, path: e.target.value }))} placeholder="路径正则" />
        <select className="border rounded px-3 text-sm" value={form.method} onChange={e => setForm(f => ({ ...f, method: e.target.value }))}><option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option><option>.*</option></select>
        <input className="border rounded px-3 text-sm" value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} placeholder="描述" />
        <Button onClick={create}><Plus className="w-4 h-4 mr-1" />新增</Button>
      </div>
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr><th className="text-left px-4 py-3">分组</th><th className="text-left px-4 py-3">名称</th><th className="text-left px-4 py-3">路径</th><th className="text-left px-4 py-3">方法</th><th className="text-right px-4 py-3">操作</th></tr></thead><tbody>
          {items.map(item => <tr key={item.id} className="border-t"><td className="px-4 py-2.5">{item.group}</td><td className="px-4 py-2.5">{item.name}</td><td className="px-4 py-2.5"><code className="text-xs">{item.path}</code></td><td className="px-4 py-2.5"><code className="text-xs">{item.method}</code></td><td className="px-4 py-2.5 text-right"><button onClick={async () => { await api.del(`/apis/${item.id}`); load() }} className="text-red-500"><Trash2 className="w-4 h-4" /></button></td></tr>)}
        </tbody></table>
      </div>
    </div>
  )
}
