import React, { useEffect, useState, useCallback } from 'react'
import { Plus, Pencil, Trash2, Loader2, Shield, Menu as MenuIcon, Route, ChevronRight } from 'lucide-react'
import { api } from '../../lib/api'
import { useI18n } from '../../contexts/i18n-context'
import { Menu, Role } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../../components/ui/dialog'

export const RolesPage: React.FC = () => {
  const { t, routeLabel } = useI18n()
  const translated = (key: string, fallback: string) => {
    const value = t(key)
    return value === key ? fallback : value
  }
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [loadFailed, setLoadFailed] = useState(false)
  const [dialog, setDialog] = useState<{ open: boolean; mode: 'create' | 'edit'; role?: Role }>({ open: false, mode: 'create' })
  const [delId, setDelId] = useState<number | null>(null)
  const [delErr, setDelErr] = useState('')
  const [form, setForm] = useState({ name: '', slug: '' })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [menus, setMenus] = useState<Menu[]>([])
  const [menuDialogRole, setMenuDialogRole] = useState<Role | null>(null)
  const [checkedMenus, setCheckedMenus] = useState<Set<number>>(new Set())
  const [policyDialogRole, setPolicyDialogRole] = useState<Role | null>(null)
  const [policyPermissionDefs, setPolicyPermissionDefs] = useState<Array<{key:string;name:string;group:string}>>([])
  const [checkedPerms, setCheckedPerms] = useState<Set<string>>(new Set())
  const [rolePerms, setRolePerms] = useState<Set<string>>(new Set())
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())

  const fetchRoles = useCallback(async () => {
    try {
      const res = await api.get<{ data: Role[] }>('/roles')
      setRoles(res.data)
    } catch {
      setLoadFailed(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchRoles() }, [fetchRoles])

  const fetchMenus = useCallback(async () => {
    try { const res = await api.get<{ data: Menu[] }>('/menus'); setMenus(res.data) } catch { setLoadFailed(true) }
  }, [])

  useEffect(() => { fetchMenus() }, [fetchMenus])

  const openCreate = () => {
    setForm({ name: '', slug: '' })
    setError('')
    setDialog({ open: true, mode: 'create' })
  }

  const openEdit = (role: Role) => {
    setForm({ name: role.name, slug: role.slug })
    setError('')
    setDialog({ open: true, mode: 'edit', role })
  }

  const handleSave = async () => {
    if (!form.name || !form.slug) { setError(t('roles.name_slug_required')); return }
    setSaving(true)
    try {
      if (dialog.mode === 'create') {
        await api.post('/roles', form)
      } else {
        await api.put(`/roles/${dialog.role!.id}`, form)
      }
      setDialog({ open: false, mode: 'create' })
      fetchRoles()
    } catch {
      setError(t('roles.failed'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!delId) return
    try {
      await api.del(`/roles/${delId}`)
      setDelId(null)
      setDelErr('')
      fetchRoles()
    } catch {
      setDelErr(t('roles.failed'))
    }
  }

  const flattenMenus = (items: Menu[], depth = 0): Array<Menu & { depth: number }> => {
    return items.flatMap(item => [
      { ...item, depth },
      ...(item.children?.length ? flattenMenus(item.children, depth + 1) : []),
    ])
  }

  const openMenuDialog = async (role: Role) => {
    const res = await api.get<{ data: number[] }>(`/roles/${role.id}/menus`)
    setCheckedMenus(new Set(res.data))
    setMenuDialogRole(role)
  }

  const saveRoleMenus = async () => {
    if (!menuDialogRole) return
    await api.put(`/roles/${menuDialogRole.id}/menus`, { menu_ids: Array.from(checkedMenus) })
    setMenuDialogRole(null)
  }

  const openPolicyDialog = async (role: Role) => {
    const res = await api.get<{ permissions: Array<{key:string;name:string;group:string}>; granted: string[] }>(`/roles/${role.slug}/permissions`)
    setPolicyPermissionDefs(res.permissions || [])
    setCheckedPerms(new Set(res.granted || []))
    setRolePerms(new Set(res.granted || []))
    setCollapsedGroups(new Set((res.permissions || []).map(p => p.group || '其他')))
    setPolicyDialogRole(role)
  }

  const togglePerm = (key: string) => {
    setCheckedPerms(prev => {
      const next = new Set(prev)
      next.has(key) ? next.delete(key) : next.add(key)
      return next
    })
  }

  const toggleGroup = (group: string) => {
    const keys = policyPermissionDefs.filter(p => (p.group || '其他') === group).map(p => p.key)
    const allChecked = keys.every(k => checkedPerms.has(k))
    setCheckedPerms(prev => {
      const next = new Set(prev)
      if (allChecked) {
        keys.forEach(k => next.delete(k))
      } else {
        keys.forEach(k => next.add(k))
      }
      return next
    })
  }

  const groupCheckState = (group: string): 'all' | 'none' | 'some' => {
    const keys = policyPermissionDefs.filter(p => (p.group || '其他') === group).map(p => p.key)
    const checked = keys.filter(k => checkedPerms.has(k)).length
    if (checked === 0) return 'none'
    if (checked === keys.length) return 'all'
    return 'some'
  }

  const toggleGroupExpand = (group: string) => {
    setCollapsedGroups(prev => {
      const next = new Set(prev)
      next.has(group) ? next.delete(group) : next.add(group)
      return next
    })
  }

  const savePolicies = async () => {
    if (!policyDialogRole) return
    await api.put(`/roles/${policyDialogRole.slug}/permissions`, { keys: Array.from(checkedPerms) })
    setPolicyDialogRole(null)
  }

  const groupedPerms = policyPermissionDefs.reduce<Record<string, typeof policyPermissionDefs>>((acc, p) => {
    const g = p.group || '其他'
    if (!acc[g]) acc[g] = []
    acc[g].push(p)
    return acc
  }, {})

  if (loading) return <div className="flex justify-center py-20" role="status" aria-label={t('common.loading')}><Loader2 className="w-6 h-6 animate-spin text-gray-400" /></div>

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-xl font-bold text-slate-800">{t('roles.title')}</h2>
        <Button size="sm" onClick={openCreate}><Plus className="w-4 h-4 mr-1" />{t('roles.create')}</Button>
      </div>
      {loadFailed && <p role="alert" className="text-red-600">{t('ui.request_failed')}</p>}

      <div className="bg-white rounded-lg border border-gray-200 overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-3 font-medium text-gray-600">ID</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('roles.name')}</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('roles.slug')}</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">{t('roles.type')}</th>
              <th className="text-right px-4 py-3 font-medium text-gray-600">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {roles.map(r => (
              <tr key={r.id} className="border-b border-gray-100 hover:bg-gray-50">
                <td className="px-4 py-2.5 text-gray-500">{r.id}</td>
                <td className="px-4 py-2.5 font-medium">{r.name}</td>
                <td className="px-4 py-2.5"><code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{r.slug}</code></td>
                <td className="px-4 py-2.5">
                  {r.is_system ? (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-amber-50 text-amber-700"><Shield className="w-3 h-3" />{t('roles.system')}</span>
                  ) : (
                    <span className="text-xs text-gray-400">{t('roles.custom')}</span>
                  )}
                </td>
                <td className="px-4 py-2.5 text-right">
                  <div className="flex items-center justify-end gap-1">
                    <button onClick={() => openMenuDialog(r)} disabled={r.is_system} className={`p-1.5 rounded text-gray-500 ${r.is_system ? 'opacity-30 cursor-not-allowed' : 'hover:bg-gray-100 hover:text-green-600'}`} title={t(r.is_system ? 'roles.system_menus' : 'roles.menu_grants')} aria-label={t(r.is_system ? 'roles.system_menus' : 'roles.menu_grants')}><MenuIcon className="w-4 h-4" /></button>
                    <button onClick={() => openPolicyDialog(r)} disabled={r.is_system} className={`p-1.5 rounded text-gray-500 ${r.is_system ? 'opacity-30 cursor-not-allowed' : 'hover:bg-gray-100 hover:text-purple-600'}`} title={t(r.is_system ? 'roles.system_permissions' : 'roles.permissions')} aria-label={t(r.is_system ? 'roles.system_permissions' : 'roles.permissions')}><Route className="w-4 h-4" /></button>
                    <button aria-label={t('roles.edit')} title={t('roles.edit')} onClick={() => openEdit(r)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-blue-600"><Pencil className="w-4 h-4" /></button>
                    <button aria-label={t('common.delete')} title={t('common.delete')} onClick={() => { setDelId(r.id); setDelErr('') }} disabled={r.is_system} className={`p-1.5 rounded text-gray-500 ${r.is_system ? 'opacity-30 cursor-not-allowed' : 'hover:bg-gray-100 hover:text-red-600'}`}><Trash2 className="w-4 h-4" /></button>
                  </div>
                </td>
              </tr>
            ))}
            {roles.length === 0 && (
              <tr><td colSpan={5} className="text-center py-8 text-gray-400">{t('common.no_data')}</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Dialog open={dialog.open} onOpenChange={v => !v && setDialog(p => ({ ...p, open: false }))}>
        <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader>
            <DialogTitle>{dialog.mode === 'create' ? t('roles.create') : t('roles.edit')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div>
              <label className="text-sm font-medium mb-1 block">{t('roles.name')}</label>
              <Input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder={t('roles.name_hint')} aria-label={t('roles.name')} />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">{t('roles.slug')}</label>
              <Input value={form.slug} onChange={e => setForm(f => ({ ...f, slug: e.target.value }))} placeholder={t('roles.slug_hint')} aria-label={t('roles.slug')} />
              <p className="text-xs text-gray-400 mt-1">{t('roles.slug_help')}</p>
            </div>
            {error && <p role="alert" className="text-red-500 text-sm">{error}</p>}
            <div className="flex flex-wrap justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setDialog({ open: false, mode: 'create' })}>{t('common.cancel')}</Button>
              <Button onClick={handleSave} disabled={saving}>{saving ? t('roles.saving') : t('common.save')}</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!delId} onOpenChange={v => !v && setDelId(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('common.confirm_delete')}</DialogTitle></DialogHeader>
          <p className="text-gray-500 text-sm py-2">{t('roles.delete_hint')}</p>
          {delErr && <p role="alert" className="text-red-500 text-sm">{delErr}</p>}
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="outline" onClick={() => { setDelId(null); setDelErr('') }}>{t('common.cancel')}</Button>
            <Button variant="destructive" onClick={handleDelete}>{t('common.confirm_delete')}</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!menuDialogRole} onOpenChange={v => !v && setMenuDialogRole(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('roles.menu_grants_title').replace('{name}', menuDialogRole?.name || '')}</DialogTitle></DialogHeader>
          <div className="mt-4 mb-6 max-h-[420px] overflow-auto border rounded-md p-2 space-y-1">
            {flattenMenus(menus).map(menu => (
              <label key={menu.id} className="flex flex-wrap items-center gap-2 text-sm rounded px-2 py-1.5 hover:bg-gray-50" style={{ paddingLeft: 8 + menu.depth * 20 }}>
                <input
                  type="checkbox"
                  checked={checkedMenus.has(menu.id)}
                  onChange={e => setCheckedMenus(prev => {
                    const next = new Set(prev)
                    e.target.checked ? next.add(menu.id) : next.delete(menu.id)
                    return next
                  })}
                />
                <span className="min-w-0 break-words">{routeLabel(menu.path, menu.name)}</span>
                {menu.path && <code className="text-xs text-gray-400">{menu.path}</code>}
              </label>
            ))}
          </div>
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="outline" onClick={() => setMenuDialogRole(null)}>{t('common.cancel')}</Button>
            <Button onClick={saveRoleMenus}>{t('roles.save_grants')}</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!policyDialogRole} onOpenChange={v => !v && setPolicyDialogRole(null)}>
        <DialogContent className="max-w-md max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('roles.permissions_title').replace('{name}', policyDialogRole?.name || '')}</DialogTitle></DialogHeader>
          <div className="mt-4 mb-6 max-h-[360px] overflow-auto border rounded-md">
            {Object.entries(groupedPerms).map(([group, items]) => {
              const gs = groupCheckState(group)
              const collapsed = collapsedGroups.has(group)
              return (
                <div key={group}>
                  <label className="flex flex-wrap items-center gap-2 px-3 py-2 bg-gray-50 border-b border-gray-100 text-xs font-medium text-gray-500 hover:bg-gray-100 cursor-pointer">
                    <button type="button" aria-label={t(collapsed ? 'roles.expand_group' : 'roles.collapse_group')} onClick={() => toggleGroupExpand(group)}
                      className="p-0.5 rounded hover:bg-gray-200">
                      <ChevronRight className={`w-3.5 h-3.5 transition-transform ${collapsed ? '' : 'rotate-90'}`} />
                    </button>
                    <input type="checkbox"
                      checked={gs === 'all'}
                      ref={el => { if (el) el.indeterminate = gs === 'some' }}
                      onChange={() => toggleGroup(group)}
                      className="shrink-0 w-4 h-4" />
                    <span className="min-w-0 break-words">{items.every(item => !item.group) ? t('roles.other_group') : translated(`permission_groups.${group}`, group)}</span>
                    <code className="text-xs text-gray-400 ml-auto break-all">{t('roles.item_count').replace('{count}', String(items.length))}</code>
                  </label>
                  {!collapsed && items.map(perm => (
                    <label key={perm.key} className="flex flex-wrap items-center gap-3 px-4 py-2 text-sm hover:bg-gray-50 border-b border-gray-100 last:border-0 cursor-pointer">
                      <input type="checkbox" checked={checkedPerms.has(perm.key)} onChange={() => togglePerm(perm.key)} className="shrink-0 w-4 h-4 ml-11" />
                      <span className="min-w-0 break-words">{translated(`permissions.${perm.key}`, perm.name)}</span>
                      <code className="text-xs text-gray-400 ml-auto break-all">{perm.key}</code>
                    </label>
                  ))}
                </div>
              )
            })}
          </div>
          <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-gray-400">
            <span>{t('roles.selected_count').replace('{count}', String(checkedPerms.size))}</span>
            <button onClick={() => checkedPerms.size > 0 && setCheckedPerms(new Set())} className="hover:text-gray-600">{t('roles.clear_selection')}</button>
          </div>
          <div className="flex flex-wrap justify-end gap-2 mt-4">
            <Button variant="outline" onClick={() => setPolicyDialogRole(null)}>{t('common.cancel')}</Button>
            <Button onClick={savePolicies}>{t('common.save')}</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
