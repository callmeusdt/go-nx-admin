import React, { useEffect, useState, useCallback } from 'react'
import { Plus, Pencil, Trash2, Wifi, WifiOff, Ban, Shield, ShieldOff, Monitor } from 'lucide-react'
import { api } from '../../lib/api'
import { useI18n } from '../../contexts/i18n-context'
import { User, Role, IPWhitelist } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Select } from '../../components/ui/select'
import { Badge } from '../../components/ui/badge'
import { PageHeader } from '../../components/ui/page-header'
import { Alert } from '../../components/ui/alert'
import { FormField } from '../../components/ui/form-field'
import { FilterBar, FilterField } from '../../components/ui/filter-bar'
import { ConfirmDialog } from '../../components/ui/confirm-dialog'
import { Tooltip } from '../../components/ui/tooltip'
import { IconButton } from '../../components/ui/icon-button'
import { SimpleTable, SimpleTableHead, SimpleTableRow, SimpleTableHeadCell, SimpleTableCell } from '../../components/ui/table'
import { Switch } from '../../components/ui/switch'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../../components/ui/dialog'
import { DataTable } from '../../components/ui/data-table'
import { useBatch, BatchSummary, batchDelete, batchSetStatus } from '../../hooks/use-batch'

export const UsersPage: React.FC = () => {
  const { t, locale } = useI18n()
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [loadFailed, setLoadFailed] = useState(false)
  const [dialog, setDialog] = useState<{ open: boolean; mode: 'create' | 'edit'; user?: User }>({ open: false, mode: 'create' })
  const [delId, setDelId] = useState<number | null>(null)
  const [currentUserId, setCurrentUserId] = useState<number | null>(null)
  const [form, setForm] = useState({ username: '', password: '', role_id: 0, status: 1, must_change_password: false })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [sessionUser, setSessionUser] = useState<User | null>(null)
  const [selected, setSelected] = useState<number[]>([])

  const [filters, setFilters] = useState({ username: '', role_id: '' })
  const [applied, setApplied] = useState(filters)

  const usersURL = useCallback(() => {
    const params = new URLSearchParams()
    if (applied.username.trim()) params.set('username', applied.username.trim())
    if (applied.role_id) params.set('role_id', applied.role_id)
    const qs = params.toString()
    return qs ? `/users?${qs}` : '/users'
  }, [applied])

  const fetchUsers = useCallback(async () => {
    try {
      const res = await api.get<{ data: User[] }>(usersURL())
      setUsers(res.data)
    } catch {
      setLoadFailed(true)
    } finally {
      setLoading(false)
    }
  }, [usersURL])

  const fetchRoles = useCallback(async () => {
    try { const res = await api.get<{ data: Role[] }>('/roles'); setRoles(res.data) } catch { setLoadFailed(true) }
  }, [])

  const fetchCurrentUser = useCallback(async () => {
    try { const res = await api.get<{ id: number }>('/auth/me'); setCurrentUserId(res.id) } catch { setLoadFailed(true) }
  }, [])

  useEffect(() => { fetchRoles(); fetchCurrentUser() }, [fetchRoles, fetchCurrentUser])
  useEffect(() => { fetchUsers() }, [fetchUsers])

  const batch = useBatch(succeeded => {
    setSelected(prev => prev.filter(id => !succeeded.includes(id)))
    fetchUsers()
  })

  const openCreate = () => {
    setForm({ username: '', password: '', role_id: roles[0]?.id || 0, status: 1, must_change_password: false })
    setError('')
    setDialog({ open: true, mode: 'create' })
  }

  const openEdit = (user: User) => {
    setForm({ username: user.username, password: '', role_id: user.role_id, status: user.status, must_change_password: !!user.must_change_password })
    setError('')
    setDialog({ open: true, mode: 'edit', user })
  }

  const handleSave = async () => {
    if (!form.username) { setError(t('users.username_required')); return }
    if (dialog.mode === 'create' && !form.password) { setError(t('users.password_required')); return }
    if (form.password && (new TextEncoder().encode(form.password).length < 8 || new TextEncoder().encode(form.password).length > 32)) { setError(t('password.policy')); return }
    setSaving(true)
    try {
      if (dialog.mode === 'create') {
        await api.post('/users', form)
      } else {
        await api.put(`/users/${dialog.user!.id}`, form)
      }
      setDialog({ open: false, mode: 'create' })
      fetchUsers()
    } catch {
      setError(t('users.failed'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!delId) return
    try {
      await api.del(`/users/${delId}`)
      setSelected(prev => prev.filter(id => id !== delId))
      setDelId(null)
      fetchUsers()
    } catch { /* */ }
  }

  const kickSession = async (sessionID: number) => {
    await api.del(`/online-users/${sessionID}`)
    const res = await api.get<{ data: User[] }>(usersURL())
    setUsers(res.data)
    setSessionUser(prev => prev ? res.data.find(item => item.id === prev.id) || null : null)
  }

  const [ipWhitelistUser, setIPWhitelistUser] = useState<User | null>(null)
  const [ipWhitelists, setIPWhitelists] = useState<IPWhitelist[]>([])
  const [ipForm, setIPForm] = useState({ ip: '', remark: '' })
  const [ipError, setIPError] = useState('')
  const [ipLoading, setIPLoading] = useState(false)

  const loadIPWhitelists = async (userID: number) => {
    const res = await api.get<{ data: IPWhitelist[] }>(`/users/${userID}/ip-whitelists`)
    setIPWhitelists(res.data)
  }

  const openIPWhitelist = (user: User) => {
    setIPWhitelistUser(user)
    setIPForm({ ip: '', remark: '' })
    setIPError('')
    loadIPWhitelists(user.id)
  }

  const saveIPWhitelist = async () => {
    if (!ipForm.ip) { setIPError(t('users.ip_required')); return }
    setIPLoading(true)
    try {
      await api.post(`/users/${ipWhitelistUser!.id}/ip-whitelists`, ipForm)
      setIPForm({ ip: '', remark: '' })
      setIPError('')
      loadIPWhitelists(ipWhitelistUser!.id)
    } catch {
      setIPError(t('users.failed'))
    } finally {
      setIPLoading(false)
    }
  }

  const toggleIPWhitelist = async (rule: IPWhitelist) => {
    await api.put(`/users/${ipWhitelistUser!.id}/ip-whitelists/${rule.id}`, {
      ip: rule.ip, remark: rule.remark, enabled: !rule.enabled,
    })
    loadIPWhitelists(ipWhitelistUser!.id)
  }

  const deleteIPWhitelist = async (ruleID: number) => {
    await api.del(`/users/${ipWhitelistUser!.id}/ip-whitelists/${ruleID}`)
    loadIPWhitelists(ipWhitelistUser!.id)
  }

  const [clearMfaUser, setClearMfaUser] = useState<User | null>(null)
  const clearMFA = async () => {
    if (!clearMfaUser) return
    await api.del(`/users/${clearMfaUser.id}/mfa`)
    setClearMfaUser(null)
    fetchUsers()
  }

  return (
    <div className="space-y-4">
      <PageHeader title={t('users.title')} actions={<Button size="sm" onClick={openCreate}><Plus className="w-4 h-4 mr-1" />{t('users.create')}</Button>} />
      {loadFailed && <Alert>{t('ui.request_failed')}</Alert>}

      <FilterBar
        onQuery={() => setApplied(filters)}
        onReset={() => { const reset = { username: '', role_id: '' }; setFilters(reset); setApplied(reset) }}
      >
        <FilterField>
          <Input
            placeholder={t('users.username')}
            aria-label={t('users.username')}
            value={filters.username}
            onChange={e => setFilters(f => ({ ...f, username: e.target.value }))}
          />
        </FilterField>
        <FilterField>
          <Select
            aria-label={t('users.role')}
            value={filters.role_id}
            onValueChange={role_id => setFilters(f => ({ ...f, role_id }))}
            options={[{ value: '', label: t('users.all_roles') }, ...roles.map(r => ({ value: String(r.id), label: r.name }))]}
          />
        </FilterField>
      </FilterBar>

      <BatchSummary text={batch.summary} />
      <DataTable<User>
        storageKey="users"
        rows={users}
        rowKey={u => u.id}
        minWidth={900}
        selectable
        selectedKeys={selected}
        onSelectionChange={keys => setSelected(keys as number[])}
        isRowDisabled={u => u.id === currentUserId}
        batchActions={({ selectedKeys }) => {
          const ids = selectedKeys as number[]
          return (
            <>
              <Button size="sm" variant="outline" onClick={() => batch.request({ labelKey: 'batch.enable', confirmKey: 'batch.confirm_enable', ...batchSetStatus('/users/status', 1) }, ids)}>{t('batch.enable')}</Button>
              <Button size="sm" variant="outline" onClick={() => batch.request({ labelKey: 'batch.disable', confirmKey: 'batch.confirm_disable', destructive: true, ...batchSetStatus('/users/status', 0) }, ids)}>{t('batch.disable')}</Button>
              <Button size="sm" variant="destructive" onClick={() => batch.request({ labelKey: 'batch.delete', confirmKey: 'batch.confirm_delete', destructive: true, ...batchDelete('/users') }, ids)}>{t('batch.delete')}</Button>
            </>
          )
        }}
        emptyText={t('common.no_data')}
        columns={[
          { key: 'id', title: 'ID', className: 'text-gray-500' },
          { key: 'username', title: t('users.username'), label: t('users.username'), className: 'font-medium' },
          { key: 'role', title: t('users.role'), label: t('users.role'), render: u => (
            <Badge variant="info">{u.role?.name || '-'}</Badge>
          ) },
          { key: 'status', title: t('users.account_status'), label: t('users.account_status'), render: u => (
            <Badge variant={u.status === 1 ? 'success' : 'danger'}>
              {u.status === 1 ? t('users.enabled') : t('users.disabled')}
            </Badge>
          ) },
          { key: 'online', title: t('users.online_status'), label: t('users.online_status'), render: u => u.is_online ? (
            <button type="button" onClick={() => setSessionUser(u)} className="rounded-full hover:opacity-80">
              <Badge variant="success" shape="pill">
                <Wifi className="w-3 h-3" />
                {t('users.online_count').replace('{count}', String(u.online_session_count || u.online_sessions?.length || 1))}
              </Badge>
            </button>
          ) : (
            <Badge shape="pill"><WifiOff className="w-3 h-3" />{t('users.offline')}</Badge>
          ) },
          { key: 'ip', title: t('users.recent_ip'), label: t('users.recent_ip'), render: u => (
            <div className="max-w-[220px]">
              <div className="font-mono text-xs text-gray-700">{u.online_ip || '-'}</div>
              {u.online_user_agent && <div className="mt-0.5 truncate text-[11px] text-gray-400" title={u.online_user_agent}>{u.online_user_agent}</div>}
            </div>
          ) },
          { key: 'created_at', title: t('users.created_at'), label: t('users.created_at'), className: 'text-gray-500 text-xs', render: u => new Date(u.created_at).toLocaleDateString(locale) },
          { key: 'actions', title: t('common.actions'), required: true, headerClassName: 'text-right', className: 'text-right', render: u => (
            <div className="flex items-center justify-end gap-1">
              <IconButton tone="primary" label={t('users.edit')} onClick={() => openEdit(u)}><Pencil className="w-4 h-4" /></IconButton>
              <IconButton tone="primary" label={t('users.ip_whitelist')} onClick={() => openIPWhitelist(u)}><Shield className="w-4 h-4" /></IconButton>
              <IconButton
                label={t(u.mfa_enabled ? 'users.clear_mfa' : 'users.mfa_disabled')}
                disabled={!u.mfa_enabled}
                onClick={() => u.mfa_enabled && setClearMfaUser(u)}
              >
                <ShieldOff className="w-4 h-4" />
              </IconButton>
              <IconButton
                tone="danger"
                label={t(currentUserId === u.id ? 'users.cannot_delete_self' : 'users.delete')}
                disabled={currentUserId === u.id}
                onClick={() => currentUserId !== u.id && setDelId(u.id)}
              >
                <Trash2 className="w-4 h-4" />
              </IconButton>
            </div>
          ) },
        ]}
      />
      {batch.dialog}

      <Dialog open={dialog.open} onOpenChange={v => !v && setDialog(p => ({ ...p, open: false }))}>
        <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader>
            <DialogTitle>{dialog.mode === 'create' ? t('users.create') : t('users.edit')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <FormField label={t('users.username')}>{id => (
              <Input
                id={id}
                value={form.username}
                onChange={e => setForm(f => ({ ...f, username: e.target.value }))}
                placeholder={t('users.username')}
                aria-label={t('users.username')}
                readOnly={dialog.mode === 'edit'}
                className={dialog.mode === 'edit' ? 'bg-gray-50 text-gray-500 cursor-not-allowed focus:ring-0' : undefined}
              />
            )}</FormField>
            <FormField label={t(dialog.mode === 'edit' ? 'users.password_unchanged' : 'users.password')}>{id => (
              <Input id={id} type="password" value={form.password} onChange={e => setForm(f => ({ ...f, password: e.target.value }))} placeholder={t('users.password')} />
            )}</FormField>
            <FormField label={t('users.role')}>{() => (
              <Select
                aria-label={t('users.role')}
                value={String(form.role_id)}
                onValueChange={v => setForm(f => ({ ...f, role_id: Number(v) }))}
                options={roles.map(r => ({ value: String(r.id), label: r.name }))}
              />
            )}</FormField>
            {dialog.mode === 'edit' && (
              <Switch
                checked={form.status === 1}
                onCheckedChange={on => setForm(f => ({ ...f, status: on ? 1 : 0 }))}
                aria-label={t('users.status')}
                label={t('users.status')}
                description={t('users.status_hint')}
              />
            )}
            <Switch
              checked={form.must_change_password}
              onCheckedChange={on => setForm(f => ({ ...f, must_change_password: on }))}
              aria-label={t('users.must_change_password')}
              label={t('users.must_change_password')}
              description={t('users.must_change_password_hint')}
            />
            {error && <Alert>{error}</Alert>}
            <div className="flex flex-wrap justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setDialog({ open: false, mode: 'create' })}>{t('common.cancel')}</Button>
              <Button onClick={handleSave} disabled={saving}>{saving ? t('users.saving') : t('common.save')}</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!delId}
        onOpenChange={v => !v && setDelId(null)}
        onConfirm={handleDelete}
        description={t('users.delete_hint')}
        destructive
      />

      <Dialog open={!!sessionUser} onOpenChange={v => !v && setSessionUser(null)}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('users.sessions_title').replace('{username}', sessionUser?.username || '')}</DialogTitle></DialogHeader>
          <SimpleTable>
            <SimpleTableHead>
              <SimpleTableRow>
                <SimpleTableHeadCell>IP</SimpleTableHeadCell>
                <SimpleTableHeadCell>User-Agent</SimpleTableHeadCell>
                <SimpleTableHeadCell>{t('users.login_at')}</SimpleTableHeadCell>
                <SimpleTableHeadCell className="text-right">{t('common.actions')}</SimpleTableHeadCell>
              </SimpleTableRow>
            </SimpleTableHead>
            <tbody>
              {sessionUser?.online_sessions?.map(session => (
                <SimpleTableRow key={session.id}>
                  <SimpleTableCell className="font-mono text-xs text-gray-700">{session.ip || '-'}</SimpleTableCell>
                  <SimpleTableCell className="max-w-[300px] truncate text-xs text-gray-500" title={session.user_agent}>{session.user_agent || '-'}</SimpleTableCell>
                  <SimpleTableCell className="text-xs text-gray-500">{new Date(session.login_at).toLocaleString(locale)}</SimpleTableCell>
                  <SimpleTableCell className="text-right">
                    {session.is_current ? (
                      <Badge variant="info">{t('users.current_session')}</Badge>
                    ) : (
                      <button
                        onClick={() => kickSession(session.id)}
                        disabled={!session.can_kick}
                        className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        <Ban className="w-3 h-3" />{t('users.kick')}
                      </button>
                    )}
                  </SimpleTableCell>
                </SimpleTableRow>
              ))}
              {!sessionUser?.online_sessions?.length && (
                <tr><td colSpan={4} className="py-8 text-center text-gray-400">{t('users.no_sessions')}</td></tr>
              )}
            </tbody>
          </SimpleTable>
        </DialogContent>
      </Dialog>

      <Dialog open={!!ipWhitelistUser} onOpenChange={v => !v && setIPWhitelistUser(null)}>
        <DialogContent className="max-w-xl max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('users.whitelist_title').replace('{username}', ipWhitelistUser?.username || '')}</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="flex flex-wrap gap-2 items-center">
              <div className="relative min-w-[180px] flex-1">
                <Input placeholder={t('users.ip_hint')} aria-label={t('users.ip_hint')} value={ipForm.ip} onChange={e => setIPForm(f => ({ ...f, ip: e.target.value }))} className="pr-9" />
                <Tooltip content={t('users.use_current_ip')}>
                  <button type="button" onClick={async () => { const r = await api.get<{ip:string}>('/auth/my-ip'); setIPForm(f => ({ ...f, ip: r.ip })) }} className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 text-gray-400 hover:text-blue-600" aria-label={t('users.use_current_ip')}>
                    <Monitor className="w-4 h-4" />
                  </button>
                </Tooltip>
              </div>
              <Input placeholder={t('users.remark')} aria-label={t('users.remark')} value={ipForm.remark} onChange={e => setIPForm(f => ({ ...f, remark: e.target.value }))} className="w-28 shrink-0" />
              <Button size="sm" aria-label={t('common.create')} onClick={saveIPWhitelist} disabled={ipLoading}><Plus className="w-4 h-4" /></Button>
            </div>
            {ipError && <Alert className="text-xs">{ipError}</Alert>}
          </div>
          <SimpleTable>
            <SimpleTableHead>
              <SimpleTableRow>
                <SimpleTableHeadCell>IP</SimpleTableHeadCell>
                <SimpleTableHeadCell>{t('users.remark')}</SimpleTableHeadCell>
                <SimpleTableHeadCell>{t('users.status')}</SimpleTableHeadCell>
                <SimpleTableHeadCell className="text-right">{t('common.actions')}</SimpleTableHeadCell>
              </SimpleTableRow>
            </SimpleTableHead>
            <tbody>
              {ipWhitelists.map(rule => (
                <SimpleTableRow key={rule.id}>
                  <SimpleTableCell><code className="text-xs">{rule.ip}</code></SimpleTableCell>
                  <SimpleTableCell className="text-xs text-gray-500">{rule.remark || '-'}</SimpleTableCell>
                  <SimpleTableCell>
                    <Switch
                      checked={rule.enabled}
                      onCheckedChange={() => toggleIPWhitelist(rule)}
                      aria-label={`${rule.ip} ${rule.enabled ? t('users.enabled') : t('users.disabled')}`}
                    />
                  </SimpleTableCell>
                  <SimpleTableCell className="text-right">
                    <button aria-label={t('common.delete')} onClick={() => deleteIPWhitelist(rule.id)} className="text-red-500 hover:text-red-700">
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </SimpleTableCell>
                </SimpleTableRow>
              ))}
              {ipWhitelists.length === 0 && (
                <tr><td colSpan={4} className="py-8 text-center text-gray-400 text-xs">{t('users.whitelist_empty')}</td></tr>
              )}
            </tbody>
          </SimpleTable>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!clearMfaUser}
        onOpenChange={v => !v && setClearMfaUser(null)}
        onConfirm={clearMFA}
        title={t('users.clear_mfa')}
        description={t('users.clear_mfa_hint').replace('{username}', clearMfaUser?.username || '')}
        confirmLabel={t('users.confirm_clear_mfa')}
        destructive
      />
    </div>
  )
}
