import React, { useEffect, useState, useCallback } from 'react'
import { Plus, Pencil, Trash2, Loader2, Wifi, WifiOff, Ban, Shield, ShieldOff, Monitor } from 'lucide-react'
import { api } from '../../lib/api'
import { User, Role, IPWhitelist } from '../../types'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../../components/ui/dialog'

export const UsersPage: React.FC = () => {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [dialog, setDialog] = useState<{ open: boolean; mode: 'create' | 'edit'; user?: User }>({ open: false, mode: 'create' })
  const [delId, setDelId] = useState<number | null>(null)
  const [currentUserId, setCurrentUserId] = useState<number | null>(null)
  const [form, setForm] = useState({ username: '', password: '', role_id: 0, status: 1 })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [sessionUser, setSessionUser] = useState<User | null>(null)

  const fetchUsers = useCallback(async () => {
    try {
      const res = await api.get<{ data: User[] }>('/users')
      setUsers(res.data)
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchRoles = useCallback(async () => {
    const res = await api.get<{ data: Role[] }>('/roles')
    setRoles(res.data)
  }, [])

  const fetchCurrentUser = useCallback(async () => {
    const res = await api.get<{ id: number }>('/auth/me')
    setCurrentUserId(res.id)
  }, [])

  useEffect(() => { fetchUsers(); fetchRoles(); fetchCurrentUser() }, [fetchUsers, fetchRoles, fetchCurrentUser])

  const openCreate = () => {
    setForm({ username: '', password: '', role_id: roles[0]?.id || 0, status: 1 })
    setError('')
    setDialog({ open: true, mode: 'create' })
  }

  const openEdit = (user: User) => {
    setForm({ username: user.username, password: '', role_id: user.role_id, status: user.status })
    setError('')
    setDialog({ open: true, mode: 'edit', user })
  }

  const handleSave = async () => {
    if (!form.username) { setError('请输入用户名'); return }
    if (dialog.mode === 'create' && !form.password) { setError('请输入密码'); return }
    setSaving(true)
    try {
      if (dialog.mode === 'create') {
        await api.post('/users', form)
      } else {
        await api.put(`/users/${dialog.user!.id}`, form)
      }
      setDialog({ open: false, mode: 'create' })
      fetchUsers()
    } catch (e: any) {
      setError(e.message)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!delId) return
    try {
      await api.del(`/users/${delId}`)
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
    if (!ipForm.ip) { setIPError('请输入 IP 地址'); return }
    setIPLoading(true)
    try {
      await api.post(`/users/${ipWhitelistUser!.id}/ip-whitelists`, ipForm)
      setIPForm({ ip: '', remark: '' })
      setIPError('')
      loadIPWhitelists(ipWhitelistUser!.id)
    } catch (e: any) {
      setIPError(e.message)
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

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-6 h-6 animate-spin text-gray-400" /></div>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-slate-800">用户管理</h2>
        <Button size="sm" onClick={openCreate}><Plus className="w-4 h-4 mr-1" />新增用户</Button>
      </div>

      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="text-left px-4 py-3 font-medium text-gray-600">ID</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">用户名</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">角色</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">账号状态</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">在线状态</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">最近 IP</th>
              <th className="text-left px-4 py-3 font-medium text-gray-600">创建时间</th>
              <th className="text-right px-4 py-3 font-medium text-gray-600">操作</th>
            </tr>
          </thead>
          <tbody>
            {users.map(u => (
              <tr key={u.id} className="border-b border-gray-100 hover:bg-gray-50">
                <td className="px-4 py-2.5 text-gray-500">{u.id}</td>
                <td className="px-4 py-2.5 font-medium">{u.username}</td>
                <td className="px-4 py-2.5">
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-blue-50 text-blue-700">
                    {u.role?.name || '-'}
                  </span>
                </td>
                <td className="px-4 py-2.5">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs ${u.status === 1 ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                    {u.status === 1 ? '启用' : '禁用'}
                  </span>
                </td>
                <td className="px-4 py-2.5">
                  {u.is_online ? (
                    <button onClick={() => setSessionUser(u)} className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-emerald-50 text-emerald-700 hover:bg-emerald-100">
                      <Wifi className="w-3 h-3" />
                      在线 {u.online_session_count || u.online_sessions?.length || 1}
                    </button>
                  ) : (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-500">
                      <WifiOff className="w-3 h-3" />离线
                    </span>
                  )}
                </td>
                <td className="px-4 py-2.5">
                  <div className="max-w-[220px]">
                    <div className="font-mono text-xs text-gray-700">{u.online_ip || '-'}</div>
                    {u.online_user_agent && <div className="mt-0.5 truncate text-[11px] text-gray-400" title={u.online_user_agent}>{u.online_user_agent}</div>}
                  </div>
                </td>
                <td className="px-4 py-2.5 text-gray-500 text-xs">{new Date(u.created_at).toLocaleDateString()}</td>
                <td className="px-4 py-2.5 text-right">
                  <div className="flex items-center justify-end gap-1">
                    <button onClick={() => openEdit(u)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-blue-600"><Pencil className="w-4 h-4" /></button>
                    <button onClick={() => openIPWhitelist(u)} className="p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-blue-600" title="IP白名单"><Shield className="w-4 h-4" /></button>
                    <button
                      onClick={() => u.mfa_enabled && setClearMfaUser(u)}
                      disabled={!u.mfa_enabled}
                      className="p-1.5 rounded text-gray-500 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent"
                      title={u.mfa_enabled ? '清空MFA' : '未启用MFA'}
                    >
                      <ShieldOff className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => currentUserId !== u.id && setDelId(u.id)}
                      disabled={currentUserId === u.id}
                      className="p-1.5 rounded text-gray-500 hover:bg-gray-100 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-500"
                      title={currentUserId === u.id ? '不能删除当前登录账户' : '删除用户'}
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {users.length === 0 && (
              <tr><td colSpan={8} className="text-center py-8 text-gray-400">暂无数据</td></tr>
            )}
          </tbody>
        </table>
      </div>

      <Dialog open={dialog.open} onOpenChange={v => !v && setDialog(p => ({ ...p, open: false }))}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{dialog.mode === 'create' ? '新增用户' : '编辑用户'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div>
              <label className="text-sm font-medium mb-1 block">用户名</label>
              <Input value={form.username} onChange={e => setForm(f => ({ ...f, username: e.target.value }))} placeholder="用户名" />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">密码{dialog.mode === 'edit' && '（留空不修改）'}</label>
              <Input type="password" value={form.password} onChange={e => setForm(f => ({ ...f, password: e.target.value }))} placeholder="密码" />
            </div>
            <div>
              <label className="text-sm font-medium mb-1 block">角色</label>
              <select
                value={form.role_id}
                onChange={e => setForm(f => ({ ...f, role_id: Number(e.target.value) }))}
                className="w-full h-10 rounded-md border border-gray-300 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {roles.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
              </select>
            </div>
            {dialog.mode === 'edit' && (
              <div>
                <label className="text-sm font-medium mb-1 block">状态</label>
                <select
                  value={form.status}
                  onChange={e => setForm(f => ({ ...f, status: Number(e.target.value) }))}
                  className="w-full h-10 rounded-md border border-gray-300 px-3 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value={1}>启用</option>
                  <option value={0}>禁用</option>
                </select>
              </div>
            )}
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
          <p className="text-gray-500 text-sm py-2">确定要删除该用户吗？此操作不可撤销。</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDelId(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>确认删除</Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!sessionUser} onOpenChange={v => !v && setSessionUser(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader><DialogTitle>{sessionUser?.username} 的在线会话</DialogTitle></DialogHeader>
          <div className="rounded-lg border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-gray-600">IP</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-600">User-Agent</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-600">登录时间</th>
                  <th className="text-right px-4 py-3 font-medium text-gray-600">操作</th>
                </tr>
              </thead>
              <tbody>
                {sessionUser?.online_sessions?.map(session => (
                  <tr key={session.id} className="border-b border-gray-100 last:border-0">
                    <td className="px-4 py-2.5 font-mono text-xs text-gray-700">{session.ip || '-'}</td>
                    <td className="px-4 py-2.5 max-w-[300px] truncate text-xs text-gray-500" title={session.user_agent}>{session.user_agent || '-'}</td>
                    <td className="px-4 py-2.5 text-xs text-gray-500">{new Date(session.login_at).toLocaleString()}</td>
                    <td className="px-4 py-2.5 text-right">
                      {session.is_current ? (
                        <span className="inline-flex rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700">当前会话</span>
                      ) : (
                        <button
                          onClick={() => kickSession(session.id)}
                          disabled={!session.can_kick}
                          className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          <Ban className="w-3 h-3" />踢下线
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
                {!sessionUser?.online_sessions?.length && (
                  <tr><td colSpan={4} className="py-8 text-center text-gray-400">暂无在线会话</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!ipWhitelistUser} onOpenChange={v => !v && setIPWhitelistUser(null)}>
        <DialogContent className="max-w-xl">
          <DialogHeader><DialogTitle>{ipWhitelistUser?.username} 的登录 IP 白名单</DialogTitle></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="flex gap-2 items-center">
              <div className="relative flex-1">
                <Input placeholder="IP 地址 (例如 192.168.1.100)" value={ipForm.ip} onChange={e => setIPForm(f => ({ ...f, ip: e.target.value }))} className="pr-9" />
                <button type="button" onClick={async () => { const r = await api.get<{ip:string}>('/auth/my-ip'); setIPForm(f => ({ ...f, ip: r.ip })) }} className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 text-gray-400 hover:text-blue-600" title="填入本机IP">
                  <Monitor className="w-4 h-4" />
                </button>
              </div>
              <Input placeholder="备注" value={ipForm.remark} onChange={e => setIPForm(f => ({ ...f, remark: e.target.value }))} className="w-28 shrink-0" />
              <Button size="sm" onClick={saveIPWhitelist} disabled={ipLoading}><Plus className="w-4 h-4" /></Button>
            </div>
            {ipError && <p className="text-red-500 text-xs">{ipError}</p>}
          </div>
          <div className="rounded-lg border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="text-left px-4 py-2 font-medium text-gray-600">IP</th>
                  <th className="text-left px-4 py-2 font-medium text-gray-600">备注</th>
                  <th className="text-left px-4 py-2 font-medium text-gray-600">状态</th>
                  <th className="text-right px-4 py-2 font-medium text-gray-600">操作</th>
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
                      <button
                        onClick={() => toggleIPWhitelist(rule)}
                        className={`inline-flex px-2 py-0.5 rounded-full text-xs ${rule.enabled ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}
                      >
                        {rule.enabled ? '启用' : '禁用'}
                      </button>
                    </td>
                    <td className="px-4 py-2 text-right">
                      <button onClick={() => deleteIPWhitelist(rule.id)} className="text-red-500 hover:text-red-700">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
                {ipWhitelists.length === 0 && (
                  <tr><td colSpan={4} className="py-8 text-center text-gray-400 text-xs">暂无白名单规则（无规则时默认放行）</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!clearMfaUser} onOpenChange={v => !v && setClearMfaUser(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>清空 MFA</DialogTitle></DialogHeader>
          <p className="text-gray-500 text-sm py-2">
            确定要清空用户 <strong>{clearMfaUser?.username}</strong> 的 MFA 设置吗？此操作将移除该用户的两步验证，且不可撤销。
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setClearMfaUser(null)}>取消</Button>
            <Button variant="destructive" onClick={clearMFA}>确认清空</Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
