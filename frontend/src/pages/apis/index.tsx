import React, { useEffect, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { ApiItem } from '../../types'
import { Button } from '../../components/ui/button'
import { useI18n } from '../../contexts/i18n-context'

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
      <div className="bg-white rounded-lg border border-gray-200 overflow-x-auto">
        <table className="w-full min-w-[640px] text-sm"><thead className="bg-gray-50"><tr><th className="text-left px-4 py-3">{t('apis.group')}</th><th className="text-left px-4 py-3">{t('apis.name')}</th><th className="text-left px-4 py-3">{t('apis.path')}</th><th className="text-left px-4 py-3">{t('apis.method')}</th><th className="text-right px-4 py-3">{t('common.actions')}</th></tr></thead><tbody>
          {items.map(item => <tr key={item.id} className="border-t"><td className="px-4 py-2.5">{groupLabel(item.group)}</td><td className="px-4 py-2.5">{nameLabel(item)}</td><td className="px-4 py-2.5"><code className="text-xs">{item.path}</code></td><td className="px-4 py-2.5"><code className="text-xs">{item.method}</code></td><td className="px-4 py-2.5 text-right"><button aria-label={t('common.delete')} onClick={() => void remove(item.id)} className="text-red-500"><Trash2 className="w-4 h-4" /></button></td></tr>)}
          {!items.length && <tr><td colSpan={5} className="p-4 text-gray-500">{t(loading ? 'common.loading' : 'common.no_data')}</td></tr>}
        </tbody></table>
      </div>
    </div>
  )
}
