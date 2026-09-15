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

export function rememberLogin(data: { token?: string; mfa_token?: string; mfa_required?: boolean; user: { username: string } }) {
  pendingMFA = !!data.mfa_required
  localStorage.setItem('username', data.user.username)
  if (cookieMode) return
  if (data.mfa_required && data.mfa_token) {
    localStorage.setItem('mfa_token', data.mfa_token)
    localStorage.setItem('mfa_username', data.user.username)
  } else if (data.token) localStorage.setItem('auth_token', data.token)
}

export function clearSession() {
  pendingMFA = false
  for (const key of ['auth_token', 'mfa_token', 'mfa_username', 'username', 'tabs', 'lock-screen']) localStorage.removeItem(key)
}

// All built-in API callers share one transport. Cookie mode never sends or
// persists a bearer token; CSRF is bound to the server-side cookie session.
export async function sessionFetch(path: string, init: RequestInit = {}) {
  const url = new URL(path, location.origin)
  if (url.origin !== location.origin || !url.pathname.startsWith('/api/v1/')) throw new Error('Invalid admin API origin')
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
  const response = await fetch(url, { ...init, headers, credentials: 'same-origin' })
  if (cookieMode && response.status === 403 && url.pathname !== '/api/v1/auth/verify-password' && !['GET','HEAD','OPTIONS'].includes((init.method || 'GET').toUpperCase())) {
    const error = await response.clone().json().catch(() => ({}))
    if (error.code === 'REAUTH_REQUIRED') {
      const verified = await new Promise<boolean>(resolve => window.dispatchEvent(new CustomEvent('nx-reauthenticate', { detail: resolve })))
      if (verified) return fetch(url, { ...init, headers, credentials: 'same-origin' })
    }
  }
  return response
}
