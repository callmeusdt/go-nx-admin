import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { sessionFetch, rememberLogin, hasPendingMFA, usesCookieSession } from '../lib/session'
import { useI18n } from '../contexts/i18n-context'

export const MFAVerifyPage: React.FC = () => {
  const { t } = useI18n()
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const mfaToken = localStorage.getItem('mfa_token')
  const username = localStorage.getItem('mfa_username')

  useEffect(() => {
    if (!hasPendingMFA() && !usesCookieSession()) {
      navigate('/login', { replace: true })
    }
  }, [mfaToken, navigate])

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!code) { setError(t('mfa.required')); return }
    setLoading(true)
    setError('')
    try {
      const res = await sessionFetch('/api/v1/auth/mfa/verify', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${mfaToken}`,
        },
        body: JSON.stringify(code.length > 6 ? { recovery_code: code } : { code }),
      })
      const data = await res.json()
      if (res.ok) {
        localStorage.removeItem('mfa_token')
        localStorage.removeItem('mfa_username')
        rememberLogin(data)
        navigate('/dashboard', { replace: true })
      } else {
        setError(t('mfa.failed'))
      }
    } catch {
      setError(t('login.network_error'))
    } finally {
      setLoading(false)
    }
  }

  const handleBack = () => {
    localStorage.removeItem('mfa_token')
    localStorage.removeItem('mfa_username')
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
      <div className="w-full max-w-sm bg-white rounded-xl shadow-2xl p-8">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-slate-900">{t('mfa.title')}</h1>
          {username ? <p className="text-slate-500 mt-2 break-all">{username}</p> : null}
          <p className="text-slate-500 mt-2">{t('mfa.hint')}</p>
        </div>
        <form onSubmit={handleVerify} className="space-y-4">
          <div>
            <input
              type="text"
              value={code}
              onChange={e => setCode(e.target.value)}
              placeholder={t('mfa.code')}
              aria-label={t('mfa.code')}
              maxLength={64}
              autoComplete="one-time-code"
              className="w-full px-4 py-2.5 rounded-lg border border-gray-300 text-center text-lg tracking-wider focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition"
            />
          </div>
          {error && <p className="text-red-500 text-sm text-center">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 rounded-lg bg-blue-600 text-white font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {loading ? t('mfa.verifying') : t('mfa.verify')}
          </button>
          <button
            type="button"
            onClick={handleBack}
            className="w-full py-2.5 rounded-lg border border-gray-300 text-gray-600 text-sm hover:bg-gray-50 transition-colors"
          >
            {t('mfa.back')}
          </button>
        </form>
      </div>
    </div>
  )
}
