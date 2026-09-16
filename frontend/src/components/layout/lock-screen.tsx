import React, { useState } from 'react'
import { useLockScreen } from '../../contexts/lock-screen-context'
import { Lock, Eye, EyeOff } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'

export const LockScreen: React.FC = () => {
  const { isLocked, unlock } = useLockScreen()
  const { t } = useI18n()
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)

  if (!isLocked) return null

  const handleUnlock = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    const ok = await unlock(password)
    setLoading(false)
    if (!ok) {
      setError(t('lock.error'))
      setPassword('')
    }
  }

  return (
    <div className="fixed inset-0 z-[9999] bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
      <div className="w-full max-w-sm px-8">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-white/10 mb-4">
            <Lock className="w-10 h-10 text-white" />
          </div>
          <h2 className="text-2xl font-bold text-white">{t('lock.title')}</h2>
          <p className="text-slate-400 mt-2">{t('lock.hint')}</p>
        </div>

        <form onSubmit={handleUnlock} className="space-y-4">
          <div className="relative">
            <input
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder={t('lock.password')}
              aria-label={t('lock.password')}
              autoFocus
              className="w-full h-10 rounded-md bg-white/10 border border-white/20 text-white placeholder:text-slate-500 px-3 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              aria-label={t(showPassword ? 'lock.hide_password' : 'lock.show_password')}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
            >
              {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          </div>

          {error && <p className="text-red-400 text-sm text-center">{error}</p>}

          <button
            type="submit"
            disabled={loading || !password}
            className="w-full py-2.5 rounded-lg bg-blue-600 text-white font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? t('lock.loading') : t('lock.unlock')}
          </button>
        </form>
      </div>
    </div>
  )
}
