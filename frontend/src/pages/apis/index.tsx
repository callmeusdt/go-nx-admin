import React, { useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { ApiItem } from '../../types'
import { Button } from '../../components/ui/button'
import { useI18n } from '../../contexts/i18n-context'
import { DataTable } from '../../components/ui/data-table'
import { useBatch, BatchSummary, batchDelete } from '../../hooks/use-batch'

const builtinNames: Record<string, [string, string]> = {
  'GET /api/v1/auth/me': ['当前用户', 'me'], 'PUT /api/v1/auth/profile': ['修改资料', 'profile'],
  'POST /api/v1/auth/logout': ['注销登录', 'logout'], 'GET /api/v1/auth/menus': ['我的菜单', 'menus'],
  'GET /api/v1/auth/my-ip': ['我的IP', 'ip'], 'POST /api/v1/auth/mfa/setup': ['配置MFA', 'setup'],
  'POST /api/v1/auth/mfa/enable': ['启用MFA', 'enable'], 'POST /api/v1/auth/mfa/disable': ['关闭MFA', 'disable'],
}

export const ApisPage: React.FC = () => {
  const { t } = useI18n()
  const [items, setItems] = useState<ApiItem[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [form, setForm] = useState({ group: '默认', name: '', path: '/api/v1/', method: 'GET', description: '' })
  const load = () => api.get<{ data: ApiItem[] }>('/apis').then(res => setItems(res.data))
    .catch(() => setError(t('apis.failed'))).finally(() => setLoading(false))
  useEffect(() => { load() }, [])
  const [selected, setSelected] = useState<number[]>([])
  const batch = useBatch(succeeded => {
    setSelected(prev => prev.filter(id => !succeeded.includes(id)))
    load()
  })

  const create = async () => {
    if (!form.name || !form.path) { setError(t('apis.required')); return }
    setError('')
    try {
      await api.post('/apis', form)
      setForm({ group: '默认', name: '', path: '/api/v1/', method: 'GET', description: '' })
      await load()
    } catch { setError(t('apis.failed')) }
  }

  const remove = async (id: number) => {
    setError('')
    try { await api.del(`/apis/${id}`); await load() }
    catch { setError(t('apis.failed')) }
  }
  const groupLabel = (group: string) => group === '默认' ? t('apis.default_group') : group === '认证' ? t('apis.auth_group') : group
  const nameLabel = (item: ApiItem) => {
    const builtin = builtinNames[`${item.method} ${item.path}`]
    return builtin && item.name === builtin[0] ? t(`apis.builtin.${builtin[1]}`) : item.name
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold text-slate-800">{t('apis.title')}</h2>
      {error && <p role="alert" className="text-red-600">{error}</p>}
      <div className="bg-white rounded-lg border border-gray-200 p-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
        <input aria-label={t('apis.group')} className="min-w-0 h-10 border rounded px-3 text-sm" value={form.group === '默认' ? t('apis.default_group') : form.group} onChange={e => setForm(f => ({ ...f, group: e.target.value }))} placeholder={t('apis.group')} />
        <input aria-label={t('apis.name')} className="min-w-0 h-10 border rounded px-3 text-sm" value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder={t('apis.name')} />
        <input aria-label={t('apis.path_pattern')} className="min-w-0 h-10 border rounded px-3 text-sm" value={form.path} onChange={e => setForm(f => ({ ...f, path: e.target.value }))} placeholder={t('apis.path_pattern')} />
        <select aria-label={t('apis.method')} className="min-w-0 h-10 border rounded px-3 text-sm" value={form.method} onChange={e => setForm(f => ({ ...f, method: e.target.value }))}><option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option><option value=".*">{t('apis.all_methods')}</option></select>
        <input aria-label={t('apis.description')} className="min-w-0 h-10 border rounded px-3 text-sm" value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} placeholder={t('apis.description')} />
        <Button onClick={create}><Plus className="w-4 h-4 mr-1" />{t('common.create')}</Button>
      </div>
      <BatchSummary text={batch.summary} />
      <DataTable<ApiItem>
        storageKey="apis"
        rows={items}
        rowKey={i => i.id}
        loading={loading}
        selectable
        selectedKeys={selected}
        onSelectionChange={keys => setSelected(keys as number[])}
        batchActions={({ selectedKeys }) => (
          <Button size="sm" variant="destructive" onClick={() => batch.request({ labelKey: 'batch.delete', confirmKey: 'batch.confirm_delete', destructive: true, ...batchDelete('/apis') }, selectedKeys as number[])}>{t('batch.delete')}</Button>
        )}
        emptyText={t('common.no_data')}
        columns={[
          { key: 'group', title: t('apis.group'), label: t('apis.group'), render: i => groupLabel(i.group) },
          { key: 'name', title: t('apis.name'), label: t('apis.name'), render: i => nameLabel(i) },
          { key: 'path', title: t('apis.path'), label: t('apis.path'), render: i => <code className="text-xs">{i.path}</code> },
          { key: 'method', title: t('apis.method'), label: t('apis.method'), render: i => <code className="text-xs">{i.method}</code> },
          { key: 'actions', title: t('common.actions'), required: true, headerClassName: 'text-right', className: 'text-right', render: i => (
            <button aria-label={t('common.delete')} onClick={() => void remove(i.id)} className="text-red-500"><Trash2 className="w-4 h-4" /></button>
          ) },
        ]}
      />
      {batch.dialog}
    </div>
  )
}
