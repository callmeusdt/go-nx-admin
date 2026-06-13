import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu'
import { Avatar, AvatarFallback } from '../ui/avatar'
import { User, Lock, LogOut, Shield, Plus, Trash2, Monitor } from 'lucide-react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '../ui/dialog'
import { Input } from '../ui/input'
import { Button } from '../ui/button'
import { useLockScreen } from '../../contexts/lock-screen-context'
import { api } from '../../lib/api'
import { IPWhitelist } from '../../types'

export const UserMenu: React.FC = () => {
  const navigate = useNavigate()
  const { lock } = useLockScreen()
  const [profileOpen, setProfileOpen] = useState(false)
  const [mfaOpen, setMfaOpen] = useState(false)
  const [username, setUsername] = useState(localStorage.getItem('username') || 'Admin')
  const [mfaStep, setMfaStep] = useState<'idle' | 'setup' | 'codes' | 'disable'>('idle')
  const [mfaCode, setMfaCode] = useState('')
  const [mfaPassword, setMfaPassword] = useState('')
  const [mfaError, setMfaError] = useState('')
  const [mfaSecret, setMfaSecret] = useState('')
  const [mfaUrl, setMfaUrl] = useState('')
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const [mfaEnabled, setMfaEnabled] = useState(false)
  const [mfaLoading, setMfaLoading] = useState(false)
  const [myWhitelists, setMyWhitelists] = useState<IPWhitelist[]>([])
  const [myIP, setMyIP] = useState('')
  const [myIPRemark, setMyIPRemark] = useState('')
  const [myIPError, setMyIPError] = useState('')

  const fetchMyWhitelists = async () => {
    try {
      const me = await api.get<{ id: number }>('/auth/me')
      const res = await api.get<{ data: IPWhitelist[] }>(`/users/${me.id}/ip-whitelists`)
      setMyWhitelists(res.data)
    } catch { /* */ }
  }

  useEffect(() => {
    if (profileOpen) fetchMyWhitelists()
  }, [profileOpen])

  const addMyIP = async () => {
    if (!myIP) { setMyIPError('请输入 IP'); return }
    try {
      const me = await api.get<{ id: number }>('/auth/me')
      await api.post(`/users/${me.id}/ip-whitelists`, { ip: myIP, remark: myIPRemark })
      setMyIP('')
      setMyIPRemark('')
      setMyIPError('')
      fetchMyWhitelists()
    } catch (e: any) {
      setMyIPError(e.message)
    }
  }

  const deleteMyIP = async (ruleID: number) => {
    const me = await api.get<{ id: number }>('/auth/me')
    await api.del(`/users/${me.id}/ip-whitelists/${ruleID}`)
    fetchMyWhitelists()
  }

  const toggleMyIP = async (rule: IPWhitelist) => {
    const me = await api.get<{ id: number }>('/auth/me')
    await api.put(`/users/${me.id}/ip-whitelists/${rule.id}`, {
      ip: rule.ip, remark: rule.remark, enabled: !rule.enabled,
    })
    fetchMyWhitelists()
  }

  const fetchMFAStatus = async () => {
    try {
      const res = await api.get<{ mfa_enabled?: boolean }>('/auth/me')
      setMfaEnabled(!!res.mfa_enabled)
    } catch { /* */ }
  }

  useEffect(() => {
    if (mfaOpen) fetchMFAStatus()
  }, [mfaOpen])

  const closeMfa = () => {
    setMfaOpen(false)
    setMfaStep('idle')
    setMfaCode('')
    setMfaPassword('')
    setMfaError('')
    setRecoveryCodes([])
  }

  const startSetup = async () => {
    setMfaLoading(true)
    try {
      const res = await api.post<{ secret: string; url: string }>('/auth/mfa/setup')
      setMfaSecret(res.secret)
      setMfaUrl(res.url)
      setMfaStep('setup')
    } catch (e: any) {
      setMfaError(e.message)
    } finally {
      setMfaLoading(false)
    }
  }

  const enableMFA = async () => {
    if (!mfaCode) { setMfaError('请输入验证码'); return }
    setMfaLoading(true)
    try {
      const res = await api.post<{ recovery_codes: string[] }>('/auth/mfa/enable', { code: mfaCode })
      setRecoveryCodes(res.recovery_codes)
      setMfaStep('codes')
    } catch (e: any) {
      setMfaError(e.message)
    } finally {
      setMfaLoading(false)
    }
  }

  const disableMFA = async () => {
    if (!mfaPassword) { setMfaError('请输入密码'); return }
    if (!mfaCode) { setMfaError('请输入验证码'); return }
    setMfaLoading(true)
    try {
      await api.post('/auth/mfa/disable', { password: mfaPassword, code: mfaCode })
      closeMfa()
    } catch (e: any) {
      setMfaError(e.message)
    } finally {
      setMfaLoading(false)
    }
  }

  const handleLogout = async () => {
    const token = localStorage.getItem('auth_token')
    if (token) {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      }).catch(() => undefined)
    }
    localStorage.removeItem('auth_token')
    localStorage.removeItem('username')
    navigate('/login', { replace: true })
  }

  const usernameInitial = username.charAt(0).toUpperCase()

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button className="flex items-center gap-2 px-3 py-1.5 rounded-lg hover:bg-gray-100 transition-colors">
            <Avatar className="w-8 h-8">
              <AvatarFallback className="bg-blue-500 text-white text-sm">
                {usernameInitial}
              </AvatarFallback>
            </Avatar>
            <span className="text-sm text-gray-700">{username}</span>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuLabel>我的账号</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => setProfileOpen(true)}>
            <User className="w-4 h-4 mr-2" />
            账号管理
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setMfaOpen(true)}>
            <Shield className="w-4 h-4 mr-2" />
            MFA 设置
          </DropdownMenuItem>
          <DropdownMenuItem onClick={lock}>
            <Lock className="w-4 h-4 mr-2" />
            锁屏
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleLogout} className="text-red-600">
            <LogOut className="w-4 h-4 mr-2" />
            注销登录
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={profileOpen} onOpenChange={setProfileOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>账号管理</DialogTitle>
            <DialogDescription>修改你的个人信息和登录安全设置</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">用户名</label>
              <Input
                value={username}
                onChange={e => setUsername(e.target.value)}
                placeholder="输入用户名"
              />
            </div>
            <div className="space-y-2">
              <p className="text-xs text-gray-400">配置后仅允许指定 IP 登录，留空则不限制</p>
              <div className="flex gap-2 items-center">
                <div className="relative flex-1">
                  <Input placeholder="IP 地址 (例如 192.168.1.100)" value={myIP} onChange={e => setMyIP(e.target.value)} className="pr-9" />
                  <button type="button" onClick={async () => { const r = await api.get<{ip:string}>('/auth/my-ip'); setMyIP(r.ip) }} className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 text-gray-400 hover:text-blue-600" title="填入本机IP">
                    <Monitor className="w-4 h-4" />
                  </button>
                </div>
                <Input placeholder="备注" value={myIPRemark} onChange={e => setMyIPRemark(e.target.value)} className="w-28 shrink-0" />
                <Button size="sm" onClick={addMyIP}><Plus className="w-4 h-4" /></Button>
              </div>
              {myIPError && <p className="text-red-500 text-xs">{myIPError}</p>}
              {myWhitelists.length > 0 && (
                <div className="rounded border border-gray-200 overflow-hidden">
                  <table className="w-full text-sm">
                    <tbody>
                      {myWhitelists.map(rule => (
                        <tr key={rule.id} className="border-b border-gray-100 last:border-0">
                          <td className="px-3 py-2"><code className="text-xs">{rule.ip}</code></td>
                          <td className="px-3 py-2 text-xs text-gray-400">{rule.remark || '-'}</td>
                          <td className="px-3 py-2">
                            <button onClick={() => toggleMyIP(rule)} className={`text-xs px-2 py-0.5 rounded-full ${rule.enabled ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                              {rule.enabled ? '启用' : '禁用'}
                            </button>
                          </td>
                          <td className="px-3 py-2 text-right">
                            <button onClick={() => deleteMyIP(rule.id)} className="text-red-500"><Trash2 className="w-3 h-3" /></button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
            <Button
              onClick={() => {
                localStorage.setItem('username', username)
                setProfileOpen(false)
              }}
            >
              保存
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={mfaOpen} onOpenChange={v => { if (!v) closeMfa() }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>MFA 两步验证</DialogTitle>
            <DialogDescription>{mfaEnabled ? '当前已启用' : '当前未启用'}</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            {mfaStep === 'idle' && (
              <div className="space-y-3">
                {mfaEnabled ? (
                  <>
                    <Button variant="destructive" onClick={() => setMfaStep('disable')} className="w-full">关闭 MFA</Button>
                    <p className="text-xs text-gray-400 text-center">关闭后登录不再需要二次验证</p>
                  </>
                ) : (
                  <Button onClick={startSetup} disabled={mfaLoading} className="w-full">
                    {mfaLoading ? '生成中...' : '启用 MFA'}
                  </Button>
                )}
              </div>
            )}
            {mfaStep === 'setup' && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">使用 Google Authenticator 或其他 TOTP 应用扫描二维码，或手动输入密钥</p>
                <div className="text-center">
                  <img
                    src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(mfaUrl)}`}
                    alt="QR Code"
                    className="mx-auto rounded-lg border w-[180px] h-[180px]"
                  />
                </div>
                <div className="text-center">
                  <code className="text-xs bg-gray-100 px-2 py-1 rounded break-all">{mfaSecret}</code>
                </div>
                <Input placeholder="输入 6 位验证码" value={mfaCode} onChange={e => setMfaCode(e.target.value)} maxLength={6} />
                {mfaError && <p className="text-red-500 text-sm">{mfaError}</p>}
                <div className="flex gap-2">
                  <Button variant="outline" onClick={closeMfa} className="flex-1">取消</Button>
                  <Button onClick={enableMFA} disabled={mfaLoading} className="flex-1">验证并启用</Button>
                </div>
              </div>
            )}
            {mfaStep === 'codes' && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">MFA 已启用。请妥善保存以下恢复码，每个恢复码仅可使用一次：</p>
                <div className="grid grid-cols-2 gap-2">
                  {recoveryCodes.map((c, i) => (
                    <code key={i} className="bg-gray-100 px-2 py-1 rounded text-xs font-mono">{c}</code>
                  ))}
                </div>
                <Button onClick={closeMfa} className="w-full">我已妥善保存</Button>
              </div>
            )}
            {mfaStep === 'disable' && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">关闭 MFA 需要验证密码和 TOTP 验证码</p>
                <Input type="password" placeholder="登录密码" value={mfaPassword} onChange={e => setMfaPassword(e.target.value)} />
                <Input placeholder="6 位验证码" value={mfaCode} onChange={e => setMfaCode(e.target.value)} maxLength={6} />
                {mfaError && <p className="text-red-500 text-sm">{mfaError}</p>}
                <div className="flex gap-2">
                  <Button variant="outline" onClick={() => { setMfaStep('idle'); setMfaError('') }} className="flex-1">返回</Button>
                  <Button variant="destructive" onClick={disableMFA} disabled={mfaLoading} className="flex-1">确认关闭</Button>
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
