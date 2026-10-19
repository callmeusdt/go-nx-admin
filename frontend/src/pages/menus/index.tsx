import React, { useEffect, useState, useCallback } from 'react'
import { Plus, Pencil, Trash2, ChevronDown, ChevronRight } from 'lucide-react'
import { api } from '../../lib/api'
import { Menu } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Select } from '../../components/ui/select'
import { Badge } from '../../components/ui/badge'
import { PageHeader } from '../../components/ui/page-header'
import { Alert } from '../../components/ui/alert'
import { Spinner } from '../../components/ui/spinner'
import { IconButton } from '../../components/ui/icon-button'
import { FormField } from '../../components/ui/form-field'
import { ConfirmDialog } from '../../components/ui/confirm-dialog'
import { DataTable } from '../../components/ui/data-table'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../../components/ui/dialog'
import { useI18n } from '../../contexts/i18n-context'

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
  const { t, routeLabel } = useI18n()
  const [menus, setMenus] = useState<Menu[]>([])
  const [flat, setFlat] = useState<FlatMenu[]>([])
  const [loading, setLoading] = useState(true)
  const [loadFailed, setLoadFailed] = useState(false)
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
    } catch {
      setLoadFailed(true)
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
    if (!form.name) { setError(t('menus.name_required')); return }
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
    } catch {
      setError(t('menus.failed'))
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

  const parentOptions = [
    { id: 0, label: t('menus.root') },
    ...flat.map(menu => ({ id: menu.id, label: routeLabel(menu.path, menu.name) })),
  ]
  const visibleFlat = visibleMenus(flat, collapsed)

  if (loading) return <Spinner label={t('common.loading')} />

  return (
    <div className="space-y-4">
      <PageHeader title={t('menus.title')} actions={<Button size="sm" onClick={() => openCreate()}><Plus aria-hidden="true" className="w-4 h-4 mr-1" />{t('menus.create')}</Button>} />
      {loadFailed && <Alert>{t('ui.request_failed')}</Alert>}

      <DataTable<Menu & { depth: number }>
        storageKey="menus"
        rows={visibleFlat}
        rowKey={m => m.id}
        minWidth={720}
        emptyText={t('common.no_data')}
        columns={[
          { key: 'name', title: t('menus.name'), label: t('menus.name'), className: 'font-medium', render: m => {
            const hasChildren = m.children && m.children.length > 0
            const isCollapsed = collapsed.has(m.id)
            return (
              <span className="flex items-center" style={{ paddingLeft: m.depth * 24 }}>
                {hasChildren ? (
                  <button onClick={() => toggleCollapse(m.id)} className="mr-1 p-0.5 rounded hover:bg-gray-200" aria-label={t(isCollapsed ? 'menus.expand' : 'menus.collapse')} title={t(isCollapsed ? 'menus.expand' : 'menus.collapse')}>
                    {isCollapsed ? <ChevronRight aria-hidden="true" className="w-3.5 h-3.5" /> : <ChevronDown aria-hidden="true" className="w-3.5 h-3.5" />}
                  </button>
                ) : (
                  <span className="w-5" />
                )}
                {routeLabel(m.path, m.name)}
              </span>
            )
          } },
          { key: 'path', title: t('menus.path'), label: t('menus.path'), className: 'text-gray-500 text-xs', render: m => m.path || '-' },
          { key: 'permission', title: t('menus.permission'), label: t('menus.permission'), render: m => <Badge className="font-mono">{m.permission || '-'}</Badge> },
          { key: 'icon', title: t('menus.icon'), label: t('menus.icon'), className: 'text-gray-500 text-xs', render: m => m.icon || '-' },
          { key: 'sort', title: t('menus.sort'), label: t('menus.sort'), className: 'text-gray-500' },
          { key: 'actions', title: t('common.actions'), required: true, headerClassName: 'text-right', className: 'text-right', render: m => (
            <div className="flex items-center justify-end gap-1">
              {!m.parent_id && (
                <IconButton tone="success" label={t('menus.add_child')} onClick={() => openCreate(m.id)}><Plus aria-hidden="true" className="w-4 h-4" /></IconButton>
              )}
              <IconButton tone="primary" label={t('common.edit')} onClick={() => openEdit(m)}><Pencil aria-hidden="true" className="w-4 h-4" /></IconButton>
              <IconButton tone="danger" label={t('common.delete')} onClick={() => setDelId(m.id)}><Trash2 aria-hidden="true" className="w-4 h-4" /></IconButton>
            </div>
          ) },
        ]}
      />

      <Dialog open={dialog.open} onOpenChange={v => !v && setDialog(p => ({ ...p, open: false }))}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{dialog.mode === 'create' ? t('menus.create') : t('menus.edit')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <FormField label={t('menus.name')}>{id => (
                <Input id={id} value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder={t('menus.name')} />
              )}</FormField>
              <FormField label={t('menus.icon')}>{id => (
                <Input id={id} value={form.icon} onChange={e => setForm(f => ({ ...f, icon: e.target.value }))} placeholder={t('menus.icon_hint')} />
              )}</FormField>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <FormField label={t('menus.route')}>{id => (
                <Input id={id} value={form.path} onChange={e => setForm(f => ({ ...f, path: e.target.value }))} placeholder="/users" />
              )}</FormField>
              <FormField label={t('menus.permission')}>{id => (
                <Input id={id} value={form.permission} onChange={e => setForm(f => ({ ...f, permission: e.target.value }))} placeholder="users:list" />
              )}</FormField>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <FormField label={t('menus.parent')}>{() => (
                <Select
                  value={form.parent_id}
                  onValueChange={parent_id => setForm(f => ({ ...f, parent_id }))}
                  aria-label={t('menus.parent')}
                  options={parentOptions.filter(o => o.id !== dialog.menu?.id).map(o => ({ value: o.id === 0 ? '' : String(o.id), label: o.label }))}
                />
              )}</FormField>
              <FormField label={t('menus.sort')}>{id => (
                <Input id={id} type="number" value={form.sort} onChange={e => setForm(f => ({ ...f, sort: Number(e.target.value) }))} />
              )}</FormField>
            </div>
            {error && <Alert>{error}</Alert>}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setDialog({ open: false, mode: 'create' })}>{t('common.cancel')}</Button>
              <Button onClick={handleSave} disabled={saving}>{saving ? t('menus.saving') : t('common.save')}</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!delId}
        onOpenChange={v => !v && setDelId(null)}
        onConfirm={handleDelete}
        description={t('menus.delete_hint')}
        destructive
      />
    </div>
  )
}
