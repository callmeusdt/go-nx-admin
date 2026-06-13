import type { DataProvider } from '@refinedev/core'

const API_URL = '/api/v1'

const authHeaders = () => {
  const token = localStorage.getItem('auth_token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

const request = async <T>(path: string, init: RequestInit = {}): Promise<T> => {
  const headers = {
    'Content-Type': 'application/json',
    ...authHeaders(),
    ...(init.headers ?? {}),
  }
  const res = await fetch(`${API_URL}${path}`, { ...init, headers })
  if (res.status === 401) {
    localStorage.removeItem('auth_token')
    localStorage.removeItem('username')
    window.location.href = '/login'
  }
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw { statusCode: res.status, message: data.message || res.statusText }
  }
  return res.json()
}

export const dataProvider: DataProvider = {
  getApiUrl: () => API_URL,
  getList: async ({ resource }) => {
    const json = await request<{ data: any[]; total?: number }>(`/${resource}`)
    return { data: json.data ?? [], total: json.total ?? json.data?.length ?? 0 }
  },
  getOne: async ({ resource, id }) => {
    const json = await request<{ data?: any }>(`/${resource}/${id}`)
    return { data: json.data ?? json }
  },
  create: async ({ resource, variables }) => {
    const json = await request<{ data?: any }>(`/${resource}`, {
      method: 'POST',
      body: JSON.stringify(variables),
    })
    return { data: json.data ?? json }
  },
  update: async ({ resource, id, variables }) => {
    const json = await request<{ data?: any }>(`/${resource}/${id}`, {
      method: 'PUT',
      body: JSON.stringify(variables),
    })
    return { data: json.data ?? json }
  },
  deleteOne: async ({ resource, id }) => {
    const json = await request<{ data?: any }>(`/${resource}/${id}`, { method: 'DELETE' })
    return { data: json.data ?? json }
  },
}
