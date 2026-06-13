import type { AuthProvider } from '@refinedev/core'

export const authProvider: AuthProvider = {
  login: async ({ username, password }) => {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    if (!res.ok) {
      const data = await res.json()
      return { success: false, error: { message: data.message || 'Login failed' } }
    }
    const data = await res.json()
    localStorage.setItem('auth_token', data.token)
    localStorage.setItem('username', data.user.username)
    return { success: true, redirectTo: '/dashboard' }
  },
  logout: async () => {
    const token = localStorage.getItem('auth_token')
    if (token) {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      }).catch(() => undefined)
    }
    localStorage.removeItem('auth_token')
    localStorage.removeItem('username')
    return { success: true, redirectTo: '/login' }
  },
  check: async () => {
    const token = localStorage.getItem('auth_token')
    if (!token) return { authenticated: false, redirectTo: '/login' }

    const res = await fetch('/api/v1/auth/me', {
      headers: { Authorization: `Bearer ${token}` },
    })
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
