import React, { useEffect, useState, useCallback } from 'react'
import { Plus, Pencil, Trash2, Loader2, Wifi, WifiOff, Ban, Shield, ShieldOff, Monitor } from 'lucide-react'
import { api } from '../../lib/api'
import { useI18n } from '../../contexts/i18n-context'
import { User, Role, IPWhitelist } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
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

  const fetchUsers = useCallback(async () => {
    try {
      const res = await api.get<{ data: User[] }>('/users')
      setUsers(res.data)
    } catch {
      setLoadFailed(true)
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchRoles = useCallback(async () => {
    try { const res = await api.get<{ data: Role[] }>('/roles'); setRoles(res.data) } catch { setLoadFailed(true) }
  }, [])

  const fetchCurrentUser = useCallback(async () => {
    try { const res = await api.get<{ id: number }>('/auth/me'); setCurrentUserId(res.id) } catch { setLoadFailed(true) }
  }, [])

  useEffect(() => { fetchUsers(); fetchRoles(); fetchCurrentUser() }, [fetchUsers, fetchRoles, fetchCurrentUser])

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
    const res = await api.get<{ data: User[] }>('/users')
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
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-xl font-bold text-slate-800">{t('users.title')}</h2>
        <Button size="sm" onClick={openCreate}><Plus className="w-4 h-4 mr-1" />{t('users.create')}</Button>
      </div>
      {loadFailed && <p role="alert" className="text-red-600">{t('ui.request_failed')}</p>}

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
            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-blue-50 text-blue-700">{u.role?.name || '-'}</span>
          ) },
          { key: 'status', title: t('users.account_status'), label: t('users.account_status'), render: u => (
            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs ${u.status === 1 ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
              {u.status === 1 ? t('users.enabled') : t('users.disabled')}
            </span>
          ) },
          { key: 'online', title: t('users.online_status'), label: t('users.online_status'), render: u => u.is_online ? (
            <button onClick={() => setSessionUser(u)} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-emerald-50 text-emerald-700 hover:bg-emerald-100">
              <Wifi className="w-3 h-3" />
              {t('users.online_count').replace('{count}', String(u.online_session_count || u.online_sessions?.length || 1))}
            </button>
          ) : (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-500">
              <WifiOff className="w-3 h-3" />{t('users.offline')}
            </span>
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
              <button aria-label={t('users.edit')} title={t('users.edit')} onClick={() => openEdit(u)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-blue-600"><Pencil className="w-4 h-4" /></button>
              <button onClick={() => openIPWhitelist(u)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-blue-600" title={t('users.ip_whitelist')} aria-label={t('users.ip_whitelist')}><Shield className="w-4 h-4" /></button>
              <button
                onClick={() => u.mfa_enabled && setClearMfaUser(u)}
                disabled={!u.mfa_enabled}
                className="p-1.5 rounded text-gray-500 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent"
                title={t(u.mfa_enabled ? 'users.clear_mfa' : 'users.mfa_disabled')}
                aria-label={t(u.mfa_enabled ? 'users.clear_mfa' : 'users.mfa_disabled')}
              >
                <ShieldOff className="w-4 h-4" />
              </button>
              <button
                onClick={() => currentUserId !== u.id && setDelId(u.id)}
                disabled={currentUserId === u.id}
                className="p-1.5 rounded text-gray-500 hover:bg-gray-100 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-500"
                title={t(currentUserId === u.id ? 'users.cannot_delete_self' : 'users.delete')}
                aria-label={t(currentUserId === u.id ? 'users.cannot_delete_self' : 'users.delete')}
              >
                <Trash2 className="w-4 h-4" />
              </button>
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
            <div>
              <label className="text-sm font-medium mb-1 block">{t('users.username')}</label>
              <Input
                value={form.username}
                onChange={e => setForm(f => ({ ...f, username: e.target.value }))}
                placeholder={t('users.username')}
                aria-label={t('users.username')}
                readOnly={dialog.mode === 'edit'}
                className={dialog.mode === 'edit' ? 'bg-gray-50 text-gray-500 cursor-not-allowed focus:ring-0' : undefined}
              />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">{t(dialog.mode === 'edit' ? 'users.password_unchanged' : 'users.password')}</label>
              <Input type="password" value={form.password} onChange={e => setForm(f => ({ ...f, password: e.target.value }))} placeholder={t('users.password')} aria-label={t('users.password')} />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">{t('users.role')}</label>
              <select
                aria-label={t('users.role')}
                value={form.role_id}
                onChange={e => setForm(f => ({ ...f, role_id: Number(e.target.value) }))}
                className="w-full h-10 rounded-md border border-gray-300 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {roles.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
              </select>
            </div>
            {dialog.mode === 'edit' && (
              <Switch
                checked={form.status === 1}
                onCheckedChange={on => setForm(f => ({ ...f, status: on ? 1 : 0 }))}
                label={t('users.status')}
                description={form.status === 1 ? t('users.enabled') : t('users.disabled')}
              />
            )}
            <Switch
              checked={form.must_change_password}
              onCheckedChange={on => setForm(f => ({ ...f, must_change_password: on }))}
              label={t('users.must_change_password')}
              description={t('users.must_change_password_hint')}
            />
            {error && <p role="alert" className="text-red-500 text-sm">{error}</p>}
            <div className="flex flex-wrap justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => setDialog({ open: false, mode: 'create' })}>{t('common.cancel')}</Button>
              <Button onClick={handleSave} disabled={saving}>{saving ? t('users.saving') : t('common.save')}</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!delId} onOpenChange={v => !v && setDelId(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('common.confirm_delete')}</DialogTitle></DialogHeader>
          <p className="text-gray-500 text-sm py-2">{t('users.delete_hint')}</p>
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="outline" onClick={() => setDelId(null)}>{t('common.cancel')}</Button>
            <Button variant="destructive" onClick={handleDelete}>{t('common.confirm_delete')}</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!sessionUser} onOpenChange={v => !v && setSessionUser(null)}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('users.sessions_title').replace('{username}', sessionUser?.username || '')}</DialogTitle></DialogHeader>
          <div className="rounded-lg border border-gray-200 overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-gray-600">IP</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-600">User-Agent</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-600">{t('users.login_at')}</th>
                  <th className="text-right px-4 py-3 font-medium text-gray-600">{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {sessionUser?.online_sessions?.map(session => (
                  <tr key={session.id} className="border-b border-gray-100 last:border-0">
                    <td className="px-4 py-2.5 font-mono text-xs text-gray-700">{session.ip || '-'}</td>
                    <td className="px-4 py-2.5 max-w-[300px] truncate text-xs text-gray-500" title={session.user_agent}>{session.user_agent || '-'}</td>
                    <td className="px-4 py-2.5 text-xs text-gray-500">{new Date(session.login_at).toLocaleString(locale)}</td>
                    <td className="px-4 py-2.5 text-right">
                      {session.is_current ? (
                        <span className="inline-flex rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700">{t('users.current_session')}</span>
                      ) : (
                        <button
                          onClick={() => kickSession(session.id)}
                          disabled={!session.can_kick}
                          className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          <Ban className="w-3 h-3" />{t('users.kick')}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
                {!sessionUser?.online_sessions?.length && (
                  <tr><td colSpan={4} className="py-8 text-center text-gray-400">{t('users.no_sessions')}</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!ipWhitelistUser} onOpenChange={v => !v && setIPWhitelistUser(null)}>
        <DialogContent className="max-w-xl max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('users.whitelist_title').replace('{username}', ipWhitelistUser?.username || '')}</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="flex flex-wrap gap-2 items-center">
              <div className="relative min-w-[180px] flex-1">
                <Input placeholder={t('users.ip_hint')} aria-label={t('users.ip_hint')} value={ipForm.ip} onChange={e => setIPForm(f => ({ ...f, ip: e.target.value }))} className="pr-9" />
                <button type="button" onClick={async () => { const r = await api.get<{ip:string}>('/auth/my-ip'); setIPForm(f => ({ ...f, ip: r.ip })) }} className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 text-gray-400 hover:text-blue-600" title={t('users.use_current_ip')} aria-label={t('users.use_current_ip')}>
                  <Monitor className="w-4 h-4" />
                </button>
              </div>
              <Input placeholder={t('users.remark')} aria-label={t('users.remark')} value={ipForm.remark} onChange={e => setIPForm(f => ({ ...f, remark: e.target.value }))} className="w-28 shrink-0" />
              <Button size="sm" aria-label={t('common.create')} onClick={saveIPWhitelist} disabled={ipLoading}><Plus className="w-4 h-4" /></Button>
            </div>
            {ipError && <p role="alert" className="text-red-500 text-xs">{ipError}</p>}
          </div>
          <div className="rounded-lg border border-gray-200 overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="text-left px-4 py-2 font-medium text-gray-600">IP</th>
                  <th className="text-left px-4 py-2 font-medium text-gray-600">{t('users.remark')}</th>
                  <th className="text-left px-4 py-2 font-medium text-gray-600">{t('users.status')}</th>
                  <th className="text-right px-4 py-2 font-medium text-gray-600">{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {ipWhitelists.map(rule => (
                  <tr key={rule.id} className="border-b border-gray-100 last:border-0">
                    <td className="px-4 py-2">
                      <code className="text-xs">{rule.ip}</code>
                    </td>
                    <td className="px-4 py-2 text-xs text-gray-500">{rule.remark || '-'}</td>
                    <td className="px-4 py-2">
                      <Switch
                        checked={rule.enabled}
                        onCheckedChange={() => toggleIPWhitelist(rule)}
                        aria-label={`${rule.ip} ${rule.enabled ? t('users.enabled') : t('users.disabled')}`}
                      />
                    </td>
                    <td className="px-4 py-2 text-right">
                      <button aria-label={t('common.delete')} onClick={() => deleteIPWhitelist(rule.id)} className="text-red-500 hover:text-red-700">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
                {ipWhitelists.length === 0 && (
                  <tr><td colSpan={4} className="py-8 text-center text-gray-400 text-xs">{t('users.whitelist_empty')}</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!clearMfaUser} onOpenChange={v => !v && setClearMfaUser(null)}>
        <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
          <DialogHeader><DialogTitle>{t('users.clear_mfa')}</DialogTitle></DialogHeader>
          <p className="text-gray-500 text-sm py-2">
            {t('users.clear_mfa_hint').replace('{username}', clearMfaUser?.username || '')}
          </p>
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="outline" onClick={() => setClearMfaUser(null)}>{t('common.cancel')}</Button>
            <Button variant="destructive" onClick={clearMFA}>{t('users.confirm_clear_mfa')}</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
