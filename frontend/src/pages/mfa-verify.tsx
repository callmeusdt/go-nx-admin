import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { sessionFetch, rememberLogin, hasPendingMFA, usesCookieSession } from '../lib/session'

export const MFAVerifyPage: React.FC = () => {
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
    if (!code) { setError('请输入验证码'); return }
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
        setError(data.message || '验证失败')
      }
    } catch {
      setError('网络错误')
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
          <h1 className="text-2xl font-bold text-slate-900">两步验证</h1>
          <p className="text-slate-500 mt-2">{username}，请输入认证器中的 6 位验证码或恢复码</p>
        </div>
        <form onSubmit={handleVerify} className="space-y-4">
          <div>
            <input
              type="text"
              value={code}
              onChange={e => setCode(e.target.value)}
              placeholder="6 位验证码 或 恢复码"
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
            {loading ? '验证中...' : '验证'}
          </button>
          <button
            type="button"
            onClick={handleBack}
            className="w-full py-2.5 rounded-lg border border-gray-300 text-gray-600 text-sm hover:bg-gray-50 transition-colors"
          >
            返回登录
          </button>
        </form>
      </div>
    </div>
  )
}
