import React, { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { SlideCaptchaModal } from '../components/ui/slide-captcha-modal'
import { useI18n } from '../contexts/i18n-context'
import { sessionFetch, rememberLogin } from '../lib/session'

interface CaptchaInfo {
  id: string
  master_img: string
  tile_img: string
  tile_width: number
  tile_height: number
  tile_y: number
}

function fpHash(): string {
  const parts = [
    navigator.userAgent,
    screen.width + 'x' + screen.height + 'x' + screen.colorDepth,
    new Date().getTimezoneOffset(),
    navigator.language,
    navigator.hardwareConcurrency || '',
  ]
  let h = 0
  const s = parts.join('|')
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) - h) + s.charCodeAt(i)
    h |= 0
  }
  return Math.abs(h).toString(36)
}

export const LoginPage: React.FC = () => {
  const { t } = useI18n()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [captchaError, setCaptchaError] = useState('')
  const [loading, setLoading] = useState(false)
  const [captchaInfo, setCaptchaInfo] = useState<CaptchaInfo | null>(null)
  const [cooldown, setCooldown] = useState(0)
  const timerRef = useRef<ReturnType<typeof setInterval>>()
  const navigate = useNavigate()
  const fp = useRef(fpHash())

  useEffect(() => {
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [])

  const startCooldown = (seconds: number, msg: string) => {
    setCaptchaError(msg)
    setCooldown(seconds)
    timerRef.current = setInterval(() => {
      setCooldown(prev => {
        const next = prev - 1
        if (next <= 0) {
          clearInterval(timerRef.current)
          setCaptchaError('')
          fetchCaptcha()
          return 0
        }
        return next
      })
    }, 1000)
  }

  const fetchCaptcha = async () => {
    try {
      const res = await sessionFetch('/api/v1/auth/captcha')
      const data = await res.json()
      if (data.captcha_enabled) {
        setCaptchaInfo({
          id: data.captcha_id,
          master_img: data.master_img,
          tile_img: data.tile_img,
          tile_width: data.tile_width,
          tile_height: data.tile_height,
          tile_y: data.tile_y || 0,
        })
      }
    } catch {
      // ignore
    }
  }

  const doLogin = async (captchaId?: string, slideX?: number) => {
    setLoading(true)
    setError('')
    setCaptchaError('')
    try {
      const body: any = { username, password, fingerprint: fp.current }
      if (captchaId) body.captcha_id = captchaId
      if (slideX !== undefined) body.slide_x = slideX
      const res = await sessionFetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const data = await res.json()

      if (data.captcha_cooldown) {
        startCooldown(data.captcha_cooldown, t('login.cooldown'))
        return
      }

      if (data.captcha_required) {
        setCaptchaInfo({
          id: data.captcha_id,
          master_img: data.master_img,
          tile_img: data.tile_img,
          tile_width: data.tile_width,
          tile_height: data.tile_height,
          tile_y: data.tile_y || 0,
        })
        setCaptchaError('')
        return
      }

      // 带 captcha 提交后关闭浮窗
      setCaptchaInfo(null)
      setCaptchaError('')

      if (res.ok) {
        rememberLogin(data)
        if (data.mfa_required) {
          navigate('/mfa-verify', { replace: true })
        } else {
          navigate('/dashboard', { replace: true })
        }
        return
      }
      setError(t('auth.login_failed'))
    } catch {
      setError(t('login.network_error'))
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    doLogin()
  }

  const handleCaptchaSubmit = (slideX: number) => {
    if (!captchaInfo) return
    doLogin(captchaInfo.id, slideX)
  }

  const handleCaptchaRefresh = () => {
    if (timerRef.current) clearInterval(timerRef.current)
    setCooldown(0)
    setCaptchaError('')
    fetchCaptcha()
  }

  const handleCaptchaCancel = () => {
    if (timerRef.current) clearInterval(timerRef.current)
    setCooldown(0)
    setCaptchaError('')
    setCaptchaInfo(null)
    setLoading(false)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center">
      <div className="w-full max-w-sm bg-white rounded-xl shadow-2xl p-8">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-slate-900">{t('login.title')}</h1>
          <p className="text-slate-500 mt-2">{t('login.subtitle')}</p>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <input type="text" value={username} onChange={e => setUsername(e.target.value)}
              placeholder={t('login.username')}
              className="w-full px-4 py-2.5 rounded-lg border border-gray-300 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition" />
          </div>
          <div>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)}
              placeholder={t('login.password')}
              className="w-full px-4 py-2.5 rounded-lg border border-gray-300 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition" />
          </div>
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button type="submit" disabled={loading}
            className="w-full py-2.5 rounded-lg bg-blue-600 text-white font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
            {loading ? t('login.loading') : t('login.submit')}
          </button>
        </form>
      </div>

      {captchaInfo && (
        <SlideCaptchaModal key={captchaInfo.id}
          captchaId={captchaInfo.id}
          masterImg={captchaInfo.master_img}
          tileImg={captchaInfo.tile_img}
          tileWidth={captchaInfo.tile_width}
          tileHeight={captchaInfo.tile_height}
          tileY={captchaInfo.tile_y}
          error={captchaError}
          cooldown={cooldown}
          onSubmit={handleCaptchaSubmit}
          onCancel={handleCaptchaCancel}
          onRefresh={handleCaptchaRefresh}
        />
      )}
    </div>
  )
}
