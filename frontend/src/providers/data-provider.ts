import type { DataProvider } from '@refinedev/core'
import { sessionFetch, clearSession } from '../lib/session'

const API_URL = '/api/v1'

const request = async <T>(path: string, init: RequestInit = {}): Promise<T> => {
  const headers = {
    'Content-Type': 'application/json',
    ...(init.headers ?? {}),
  }
  const res = await sessionFetch(`${API_URL}${path}`, { ...init, headers })
  if (res.status === 401) {
    clearSession()
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
  getList: async ({ resource, pagination, filters, sorters }) => {
    const query = new URLSearchParams({ page: String(pagination?.current || 1), page_size: String(pagination?.pageSize || 20) })
    for (const filter of filters || []) if ('field' in filter && filter.value != null) query.set(filter.field, String(filter.value))
    if (sorters?.length) { query.set('sort', sorters[0].field); query.set('order', sorters[0].order) }
    const json = await request<{ data: any[]; total?: number }>(`/${resource}?${query}`)
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
