import React, { useState } from 'react'
import { sessionFetch } from '../../lib/session'
import { useI18n } from '../../contexts/i18n-context'
import { Input } from '../ui/input'
import { Button } from '../ui/button'

const bytes = (value: string) => new TextEncoder().encode(value).length

// Server codes -> message keys. Anything unknown falls back to a generic error.
const ERROR_KEYS: Record<string, string> = {
  OLD_PASSWORD_INCORRECT: 'password.err_old',
  MFA_CODE_INVALID: 'password.err_code',
  PASSWORD_POLICY: 'password.policy',
  PASSWORD_UNCHANGED: 'password.err_same',
  TOO_MANY_ATTEMPTS: 'password.err_limited',
}

interface Props {
  mfaEnabled: boolean
  onChanged: () => void
  onCancel?: () => void
  cancelLabel?: string
}

/** Shared by the security settings dialog and the forced-change gate. */
export const PasswordForm: React.FC<Props> = ({ mfaEnabled, onChanged, onCancel, cancelLabel }) => {
  const { t } = useI18n()
  const [oldPassword, setOld] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (busy) return
    setError('')
    if (!oldPassword) { setError(t('password.err_old_required')); return }
    if (bytes(next) < 12 || bytes(next) > 72) { setError(t('password.policy')); return }
    if (next !== confirm) { setError(t('password.err_mismatch')); return }
    if (next === oldPassword) { setError(t('password.err_same')); return }
    if (mfaEnabled && !code) { setError(t('account.mfa_required')); return }
    setBusy(true)
    try {
      const res = await sessionFetch('/api/v1/auth/password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ old_password: oldPassword, new_password: next, code }),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => null)
        setError(t(ERROR_KEYS[data?.code] ?? 'ui.request_failed'))
        return
      }
      setOld(''); setNext(''); setConfirm(''); setCode('')
      onChanged()
    } catch {
      setError(t('ui.request_failed'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form className="space-y-3" onSubmit={submit}>
      <Input type="password" autoComplete="current-password" aria-label={t('password.old')} placeholder={t('password.old')} value={oldPassword} onChange={e => setOld(e.target.value)} />
      <Input type="password" autoComplete="new-password" aria-label={t('password.new')} placeholder={t('password.new')} value={next} onChange={e => setNext(e.target.value)} />
      <Input type="password" autoComplete="new-password" aria-label={t('password.confirm')} placeholder={t('password.confirm')} value={confirm} onChange={e => setConfirm(e.target.value)} />
      {mfaEnabled && (
        <Input autoComplete="one-time-code" inputMode="numeric" maxLength={6} aria-label={t('account.mfa_code')} placeholder={t('account.mfa_code')} value={code} onChange={e => setCode(e.target.value)} />
      )}
      <p className="text-xs text-gray-400">{t('password.policy')}</p>
      {error && <p role="alert" className="text-red-500 text-sm">{error}</p>}
      <div className="flex justify-end gap-2">
        {onCancel && <Button type="button" variant="outline" onClick={onCancel} disabled={busy}>{cancelLabel ?? t('common.cancel')}</Button>}
        <Button type="submit" disabled={busy}>{busy ? t('users.saving') : t('password.submit')}</Button>
      </div>
    </form>
  )
}
