import React, { useCallback, useEffect, useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '../ui/dialog'
import { api } from '../../lib/api'
import { useI18n } from '../../contexts/i18n-context'
import { PasswordForm } from './password-form'
import { sessionFetch, clearSession } from '../../lib/session'

interface Me { must_change_password?: boolean; mfa_enabled?: boolean }

/**
 * While the account is flagged must_change_password the server refuses every
 * other API call, so rendering the app would only show failed requests. In that
 * state show nothing but a dialog that cannot be dismissed: the only ways out
 * are changing the password or signing out.
 */
export const PasswordGate: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { t } = useI18n()
  const [me, setMe] = useState<Me | null>(null)

  const load = useCallback(() => {
    api.get<Me>('/auth/me').then(setMe).catch(() => { /* a 401 is handled by the api layer */ })
  }, [])
  useEffect(() => { load() }, [load])

  if (!me?.must_change_password) return <>{children}</>

  const signOut = async () => {
    await sessionFetch('/api/v1/auth/logout', { method: 'POST' }).catch(() => null)
    clearSession()
    window.location.assign('/login')
  }

  return (
    <div className="h-screen bg-gray-100">
      <Dialog open onOpenChange={() => { /* not dismissible */ }}>
        <DialogContent hideClose className="max-w-md" onEscapeKeyDown={e => e.preventDefault()} onPointerDownOutside={e => e.preventDefault()} onInteractOutside={e => e.preventDefault()}>
          <DialogHeader className="mb-4">
            <DialogTitle>{t('password.force_title')}</DialogTitle>
            <DialogDescription>{t('password.force_hint')}</DialogDescription>
          </DialogHeader>
          <PasswordForm
            mfaEnabled={!!me.mfa_enabled}
            onChanged={() => window.location.reload()}
            onCancel={signOut}
            cancelLabel={t('password.signout')}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
