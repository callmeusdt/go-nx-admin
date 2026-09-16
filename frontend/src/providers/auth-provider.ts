import type { AuthProvider } from '@refinedev/core'
import { sessionFetch, rememberLogin, clearSession } from '../lib/session'
import { builtinMessage } from '../contexts/i18n-context'

export const authProvider: AuthProvider = {
  login: async ({ username, password }) => {
    const res = await sessionFetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    if (!res.ok) {
      await res.json().catch(() => null)
      return { success: false, error: new Error(builtinMessage(document.documentElement.lang, 'auth.login_failed')) }
    }
    const data = await res.json()
    rememberLogin(data)
    return { success: true, redirectTo: data.mfa_required ? '/mfa-verify' : '/dashboard' }
  },
  logout: async () => {
    const result = await sessionFetch('/api/v1/auth/logout', { method: 'POST' })
    if (!result.ok && result.status !== 401) return { success: false, error: new Error(builtinMessage(document.documentElement.lang, 'ui.request_failed')) }
    clearSession()
    return { success: true, redirectTo: '/login' }
  },
  check: async () => {
    const res = await sessionFetch('/api/v1/auth/me')
    if (res.ok) return { authenticated: true }

    localStorage.removeItem('auth_token')
    localStorage.removeItem('username')
    return { authenticated: false, redirectTo: '/login' }
  },
  onError: async (error) => {
    console.error(error)
    return { error }
  },
  getPermissions: async () => null,
  getIdentity: async () => {
    const username = localStorage.getItem('username')
    if (!username) return null
    return { name: username }
  },
}
