import { resetPermissions } from '../dashboard/permissions'

let cookieMode = false
let pendingMFA = false

export function configureSession(mode: 'bearer' | 'cookie' = 'bearer') {
  cookieMode = mode === 'cookie'
  if (cookieMode) {
    localStorage.removeItem('auth_token')
    localStorage.removeItem('mfa_token')
  }
}
export const usesCookieSession = () => cookieMode
export const hasPendingMFA = () => cookieMode ? pendingMFA : !!localStorage.getItem('mfa_token')

export function rememberLogin(data: { token?: string; refresh_token?: string; mfa_token?: string; mfa_required?: boolean; user: { username: string } }) {
  resetPermissions()
  pendingMFA = !!data.mfa_required
  localStorage.setItem('username', data.user.username)
  if (cookieMode) return
  if (data.mfa_required && data.mfa_token) {
    localStorage.setItem('mfa_token', data.mfa_token)
    localStorage.setItem('mfa_username', data.user.username)
  } else if (data.token) {
    localStorage.setItem('auth_token', data.token)
    if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token)
    else localStorage.removeItem('refresh_token')
  }
}

export function clearSession() {
  resetPermissions()
  pendingMFA = false
  for (const key of ['auth_token', 'refresh_token', 'mfa_token', 'mfa_username', 'username', 'tabs', 'lock-screen']) localStorage.removeItem(key)
}

// Exchanges the stored refresh token for a new pair. Concurrent callers share
// one in-flight request: the server rotates the token, so a second request with
// the same value would be treated as a replay.
let refreshing: Promise<boolean> | null = null

async function refreshAccessToken(): Promise<boolean> {
  const refresh = localStorage.getItem('refresh_token')
  if (!refresh) return false
  refreshing ??= (async () => {
    try {
      const res = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refresh }),
        credentials: 'same-origin',
      })
      if (!res.ok) return false
      const data = await res.json()
      if (!data.token) return false
      localStorage.setItem('auth_token', data.token)
      if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token)
      return true
    } catch {
      return false
    } finally {
      refreshing = null
    }
  })()
  return refreshing
}

const NO_REFRESH_PATHS = ['/api/v1/auth/login', '/api/v1/auth/refresh', '/api/v1/auth/mfa/verify', '/api/v1/auth/captcha']

// All built-in API callers share one transport. Cookie mode never sends or
// persists a bearer token; CSRF is bound to the server-side cookie session.
export async function sessionFetch(path: string, init: RequestInit = {}) {
  const url = new URL(path, location.origin)
  if (url.origin !== location.origin || !url.pathname.startsWith('/api/v1/')) throw new Error('Invalid admin API origin')
  const explicitAuth = new Headers(init.headers).has('Authorization')
  const send = () => {
    const headers = new Headers(init.headers)
    if (cookieMode) {
      headers.delete('Authorization')
      if (!['GET', 'HEAD', 'OPTIONS'].includes((init.method || 'GET').toUpperCase())) {
        const csrf = document.cookie.split('; ').find(value => value.startsWith('__Host-nx-csrf='))?.slice('__Host-nx-csrf='.length)
        if (csrf) headers.set('X-CSRF-Token', decodeURIComponent(csrf))
      }
    } else if (!headers.has('Authorization')) {
      const token = localStorage.getItem('auth_token')
      if (token) headers.set('Authorization', `Bearer ${token}`)
    }
    return { headers, request: fetch(url, { ...init, headers, credentials: 'same-origin' }) }
  }
  let { headers, request } = send()
  let response = await request
  // An expired access token is recoverable while the refresh token is valid.
  if (!cookieMode && !explicitAuth && response.status === 401 && !NO_REFRESH_PATHS.includes(url.pathname) && await refreshAccessToken()) {
    ;({ headers, request } = send())
    response = await request
  }
  if (cookieMode && response.status === 403 && url.pathname !== '/api/v1/auth/verify-password' && !['GET','HEAD','OPTIONS'].includes((init.method || 'GET').toUpperCase())) {
    const error = await response.clone().json().catch(() => ({}))
    if (error.code === 'REAUTH_REQUIRED') {
      const verified = await new Promise<boolean>(resolve => window.dispatchEvent(new CustomEvent('nx-reauthenticate', { detail: resolve })))
      if (verified) return fetch(url, { ...init, headers, credentials: 'same-origin' })
    }
  }
  return response
}
