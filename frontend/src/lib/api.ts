import { sessionFetch, clearSession } from './session'
const BASE = '/api/v1'

async function request<T = any>(
  path: string,
  method: string,
  body?: any,
): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }

  const res = await sessionFetch(`${BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (res.status === 401) {
    clearSession()
    window.location.href = '/login'
    throw new Error('登录已失效，请重新登录')
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: 'request failed' }))
    throw new Error(err.message || 'request failed')
  }

  return res.json()
}

export const api = {
  get: <T = any>(path: string) => request<T>(path, 'GET'),
  post: <T = any>(path: string, body?: any) => request<T>(path, 'POST', body),
  put: <T = any>(path: string, body?: any) => request<T>(path, 'PUT', body),
  del: <T = any>(path: string) => request<T>(path, 'DELETE'),
}
