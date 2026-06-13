import React, { useEffect, useState, useCallback } from 'react'
import { Plus, Pencil, Trash2, Loader2, ChevronDown, ChevronRight } from 'lucide-react'
import { api } from '../../lib/api'
import { Menu } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../../components/ui/dialog'

interface FlatMenu {
  id: number
  parent_id: number | null
  path: string
  permission: string
  name: string
  icon: string
  sort: number
  children?: Menu[]
  depth: number
  parentPath: number[]
}

function flattenMenus(menus: Menu[], depth = 0, parentPath: number[] = []): FlatMenu[] {
  const result: FlatMenu[] = []
  for (const m of menus) {
    const { children, ...rest } = m
    result.push({ ...rest, depth, parentPath, children: m.children })
    if (m.children?.length) {
      result.push(...flattenMenus(m.children, depth + 1, [...parentPath, m.id]))
    }
  }
  return result
}

function visibleMenus(flat: FlatMenu[], collapsed: Set<number>) {
  return flat.filter(item => !item.parentPath.some(id => collapsed.has(id)))
}

export const MenusPage: React.FC = () => {
  const [menus, setMenus] = useState<Menu[]>([])
  const [flat, setFlat] = useState<FlatMenu[]>([])
  const [loading, setLoading] = useState(true)
  const [collapsed, setCollapsed] = useState<Set<number>>(new Set())
  const [dialog, setDialog] = useState<{ open: boolean; mode: 'create' | 'edit'; menu?: Menu }>({ open: false, mode: 'create' })
  const [delId, setDelId] = useState<number | null>(null)
  const [form, setForm] = useState({ name: '', path: '', icon: '', permission: '', parent_id: '', sort: 0 })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const fetchMenus = useCallback(async () => {
    try {
      const res = await api.get<{ data: Menu[] }>('/menus')
      setMenus(res.data)
      setFlat(flattenMenus(res.data))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchMenus() }, [fetchMenus])

  const toggleCollapse = (id: number) => {
    setCollapsed(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  const openCreate = (parentId?: number) => {
    setForm({ name: '', path: '', icon: '', permission: '', parent_id: parentId ? String(parentId) : '', sort: 0 })
    setError('')
    setDialog({ open: true, mode: 'create' })
  }

  const openEdit = (menu: Menu) => {
    setForm({
      name: menu.name,
      path: menu.path || '',
      icon: menu.icon || '',
      permission: menu.permission || '',
      parent_id: menu.parent_id ? String(menu.parent_id) : '',
      sort: menu.sort,
    })
    setError('')
    setDialog({ open: true, mode: 'edit', menu })
  }

  const handleSave = async () => {
    if (!form.name) { setError('菜单名称不能为空'); return }
    const body = {
      ...form,
      parent_id: form.parent_id ? Number(form.parent_id) : null,
    }
    setSaving(true)
    try {
      if (dialog.mode === 'create') {
        await api.post('/menus', body)
      } else {
        await api.put(`/menus/${dialog.menu!.id}`, body)
      }
      setDialog({ open: false, mode: 'create' })
      fetchMenus()
    } catch (e: any) {
      setError(e.message)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!delId) return
    try {
      await api.del(`/menus/${delId}`)
      setDelId(null)
      fetchMenus()
    } catch { /* */ }
  }

  const parentOptions = [{ id: 0, name: '根目录' }, ...flat]
  const visibleFlat = visibleMenus(flat, collapsed)

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-6 h-6 animate-spin text-gray-400" /></div>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-slate-800">菜单管理</h2>
        <Button size="sm" onClick={() => openCreate()}><Plus className="w-4 h-4 mr-1" />新增菜单</Button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-3 font-medium text-gray-600">菜单名称</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">路径</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">权限标识</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">图标</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">排序</th>
              <th className="text-right px-4 py-3 font-medium text-gray-600">操作</th>
            </tr>
          </thead>
          <tbody>
            {visibleFlat.map(m => {
              const hasChildren = m.children && m.children.length > 0
              const isCollapsed = collapsed.has(m.id)
              return (
                <tr key={m.id} className="border-b border-gray-100 hover:bg-gray-50">
                  <td className="px-4 py-2.5 font-medium">
                    <span className="flex items-center" style={{ paddingLeft: m.depth * 24 }}>
                      {hasChildren ? (
                        <button onClick={() => toggleCollapse(m.id)} className="mr-1 p-0.5 rounded hover:bg-gray-200">
                          {isCollapsed ? <ChevronRight className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
                        </button>
                      ) : (
                        <span className="w-5" />
                      )}
                      {m.name}
                    </span>
                  </td>
                  <td className="px-4 py-2.5 text-gray-500 text-xs">{m.path || '-'}</td>
                  <td className="px-4 py-2.5"><code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{m.permission || '-'}</code></td>
                  <td className="px-4 py-2.5 text-gray-500 text-xs">{m.icon || '-'}</td>
                  <td className="px-4 py-2.5 text-gray-500">{m.sort}</td>
                  <td className="px-4 py-2.5 text-right">
                    <div className="flex items-center justify-end gap-1">
                      {!m.parent_id && (
                        <button onClick={() => openCreate(m.id)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-green-600" title="添加子菜单"><Plus className="w-4 h-4" /></button>
                      )}
                      <button onClick={() => openEdit(m)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-blue-600"><Pencil className="w-4 h-4" /></button>
                      <button onClick={() => setDelId(m.id)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-red-600"><Trash2 className="w-4 h-4" /></button>
                    </div>
                  </td>
                </tr>
              )
            })}
            {visibleFlat.length === 0 && (
              <tr><td colSpan={6} className="text-center py-8 text-gray-400">暂无数据</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Dialog open={dialog.open} onOpenChange={v => !v && setDialog(p => ({ ...p, open: false }))}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{dialog.mode === 'create' ? '新增菜单' : '编辑菜单'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-sm font-medium mb-1 block">菜单名称</label>
                <Input value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="菜单名称" />
              </div>
              <div>
                <label className="text-sm font-medium mb-1 block">图标</label>
                <Input value={form.icon} onChange={e => setForm(f => ({ ...f, icon: e.target.value }))} placeholder="lucide图标名" />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-sm font-medium mb-1 block">路由路径</label>
                <Input value={form.path} onChange={e => setForm(f => ({ ...f, path: e.target.value }))} placeholder="/users" />
              </div>
              <div>
                <label className="text-sm font-medium mb-1 block">权限标识</label>
                <Input value={form.permission} onChange={e => setForm(f => ({ ...f, permission: e.target.value }))} placeholder="users:list" />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-sm font-medium mb-1 block">父级菜单</label>
                <select
                  value={form.parent_id}
                  onChange={e => setForm(f => ({ ...f, parent_id: e.target.value }))}
                  className="w-full h-10 rounded-md border border-gray-300 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {parentOptions.filter(o => o.id !== dialog.menu?.id).map(o => (
                    <option key={o.id} value={o.id === 0 ? '' : o.id}>{o.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium mb-1 block">排序</label>
                <Input type="number" value={form.sort} onChange={e => setForm(f => ({ ...f, sort: Number(e.target.value) }))} />
              </div>
            </div>
            {error && <p className="text-red-500 text-sm">{error}</p>}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setDialog({ open: false, mode: 'create' })}>取消</Button>
              <Button onClick={handleSave} disabled={saving}>{saving ? '保存中...' : '保存'}</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!delId} onOpenChange={v => !v && setDelId(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>确认删除</DialogTitle></DialogHeader>
          <p className="text-gray-500 text-sm py-2">确定要删除该菜单吗？子菜单也会被级联删除。</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDelId(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
