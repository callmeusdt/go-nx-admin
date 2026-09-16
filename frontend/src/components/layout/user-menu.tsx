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
import { sessionFetch, clearSession } from '../../lib/session'
import { IPWhitelist } from '../../types'
import { useI18n } from '../../contexts/i18n-context'

export const UserMenu: React.FC = () => {
  const { t } = useI18n()
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
    if (!myIP) { setMyIPError(t('account.ip_required')); return }
    try {
      const me = await api.get<{ id: number }>('/auth/me')
      await api.post(`/users/${me.id}/ip-whitelists`, { ip: myIP, remark: myIPRemark })
      setMyIP('')
      setMyIPRemark('')
      setMyIPError('')
      fetchMyWhitelists()
    } catch (e: any) {
      setMyIPError(t('account.ip_error'))
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
    setMfaSecret('')
  }

  const startSetup = async () => {
    setMfaLoading(true)
    setMfaError('')
    try {
      const res = await api.post<{ secret: string; url: string }>('/auth/mfa/setup')
      setMfaSecret(res.secret)
      setMfaStep('setup')
    } catch (e: any) {
      setMfaError(t('account.mfa_error'))
    } finally {
      setMfaLoading(false)
    }
  }

  const enableMFA = async () => {
    if (!mfaCode) { setMfaError(t('account.mfa_required')); return }
    setMfaLoading(true)
    setMfaError('')
    try {
      const res = await api.post<{ recovery_codes: string[] }>('/auth/mfa/enable', { code: mfaCode })
      setRecoveryCodes(res.recovery_codes)
      setMfaStep('codes')
    } catch (e: any) {
      setMfaError(t('account.mfa_error'))
    } finally {
      setMfaLoading(false)
    }
  }

  const disableMFA = async () => {
    if (!mfaPassword) { setMfaError(t('account.password_required')); return }
    if (!mfaCode) { setMfaError(t('account.mfa_required')); return }
    setMfaLoading(true)
    setMfaError('')
    try {
      await api.post('/auth/mfa/disable', { password: mfaPassword, code: mfaCode })
      closeMfa()
    } catch (e: any) {
      setMfaError(t('account.mfa_error'))
    } finally {
      setMfaLoading(false)
    }
  }

  const handleLogout = async () => {
    const response = await sessionFetch('/api/v1/auth/logout', { method: 'POST' })
    if (!response.ok && response.status !== 401) return
    clearSession()
    navigate('/login', { replace: true })
  }

  const usernameInitial = username.charAt(0).toUpperCase()

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button aria-label={username} className="flex min-w-0 items-center gap-2 px-2 sm:px-3 py-1.5 rounded-lg hover:bg-gray-100 transition-colors">
            <Avatar className="w-8 h-8">
              <AvatarFallback className="bg-blue-500 text-white text-sm">
                {usernameInitial}
              </AvatarFallback>
            </Avatar>
            <span className="hidden sm:block max-w-40 truncate text-sm text-gray-700">{username}</span>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuLabel>{t('account.title')}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => setProfileOpen(true)}>
            <User className="w-4 h-4 mr-2" />
            {t('account.manage')}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setMfaOpen(true)}>
            <Shield className="w-4 h-4 mr-2" />
            {t('account.mfa_settings')}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={lock}>
            <Lock className="w-4 h-4 mr-2" />
            {t('account.lock')}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleLogout} className="text-red-600">
            <LogOut className="w-4 h-4 mr-2" />
            {t('account.logout')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={profileOpen} onOpenChange={setProfileOpen}>
        <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{t('account.manage')}</DialogTitle>
            <DialogDescription>{t('account.hint')}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('account.username')}</label>
              <Input
                value={username}
                onChange={e => setUsername(e.target.value)}
                placeholder={t('account.username_placeholder')}
              />
            </div>
            <div className="space-y-2">
              <p className="text-xs text-gray-400">{t('account.ip_hint')}</p>
              <div className="flex flex-wrap gap-2 items-center">
                <div className="relative min-w-40 flex-1">
                  <Input placeholder={t('account.ip_placeholder')} value={myIP} onChange={e => setMyIP(e.target.value)} className="pr-9" />
                  <button type="button" onClick={async () => { const r = await api.get<{ip:string}>('/auth/my-ip'); setMyIP(r.ip) }} className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 text-gray-400 hover:text-blue-600" title={t('account.current_ip')}>
                    <Monitor className="w-4 h-4" />
                  </button>
                </div>
                <Input placeholder={t('account.remark')} value={myIPRemark} onChange={e => setMyIPRemark(e.target.value)} className="w-28 shrink-0" />
                <Button size="sm" aria-label={t('account.add_ip')} onClick={addMyIP}><Plus className="w-4 h-4" /></Button>
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
                              {rule.enabled ? t('account.enabled') : t('account.disabled')}
                            </button>
                          </td>
                          <td className="px-3 py-2 text-right">
                            <button aria-label={t('account.delete_ip')} onClick={() => deleteMyIP(rule.id)} className="text-red-500"><Trash2 className="w-3 h-3" /></button>
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
              {t('account.save')}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={mfaOpen} onOpenChange={v => { if (!v) closeMfa() }}>
        <DialogContent className="max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{t('account.mfa_title')}</DialogTitle>
            <DialogDescription>{mfaEnabled ? t('account.mfa_enabled') : t('account.mfa_disabled')}</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            {mfaStep === 'idle' && mfaError && <p role="alert" className="text-red-500 text-sm mb-3">{mfaError}</p>}
            {mfaStep === 'idle' && (
              <div className="space-y-3">
                {mfaEnabled ? (
                  <>
                    <Button variant="destructive" onClick={() => setMfaStep('disable')} className="w-full">{t('account.mfa_disable')}</Button>
                    <p className="text-xs text-gray-400 text-center">{t('account.mfa_disable_hint')}</p>
                  </>
                ) : (
                  <Button onClick={startSetup} disabled={mfaLoading} className="w-full">
                    {mfaLoading ? t('account.mfa_generating') : t('account.mfa_enable')}
                  </Button>
                )}
              </div>
            )}
            {mfaStep === 'setup' && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">{t('account.mfa_setup_hint')}</p>
                <div className="text-center">
                  <code aria-label={t('account.mfa_secret')} className="text-xs bg-gray-100 px-2 py-1 rounded break-all">{mfaSecret}</code>
                </div>
                <Input placeholder={t('account.mfa_code')} value={mfaCode} onChange={e => setMfaCode(e.target.value)} maxLength={6} />
                {mfaError && <p className="text-red-500 text-sm">{mfaError}</p>}
                <div className="flex gap-2">
                  <Button variant="outline" onClick={closeMfa} className="flex-1">{t('account.cancel')}</Button>
                  <Button onClick={enableMFA} disabled={mfaLoading} className="flex-1">{t('account.mfa_verify_enable')}</Button>
                </div>
              </div>
            )}
            {mfaStep === 'codes' && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">{t('account.mfa_codes_hint')}</p>
                <div className="grid grid-cols-2 gap-2">
                  {recoveryCodes.map((c, i) => (
                    <code key={i} className="bg-gray-100 px-2 py-1 rounded text-xs font-mono">{c}</code>
                  ))}
                </div>
                <Button onClick={closeMfa} className="w-full">{t('account.mfa_saved')}</Button>
              </div>
            )}
            {mfaStep === 'disable' && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">{t('account.mfa_disable_verify')}</p>
                <Input type="password" placeholder={t('account.password')} value={mfaPassword} onChange={e => setMfaPassword(e.target.value)} />
                <Input placeholder={t('account.mfa_code')} value={mfaCode} onChange={e => setMfaCode(e.target.value)} maxLength={6} />
                {mfaError && <p className="text-red-500 text-sm">{mfaError}</p>}
                <div className="flex gap-2">
                  <Button variant="outline" onClick={() => { setMfaStep('idle'); setMfaError('') }} className="flex-1">{t('account.back')}</Button>
                  <Button variant="destructive" onClick={disableMFA} disabled={mfaLoading} className="flex-1">{t('account.mfa_confirm_disable')}</Button>
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
