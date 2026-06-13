import simpleRestProvider from '@refinedev/simple-rest'

const API_URL = '/api/v1'

const baseProvider = simpleRestProvider(API_URL)

const authHeaders = () => {
  const token = localStorage.getItem('auth_token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

const withAuthRedirect = async <T>(promise: Promise<T>): Promise<T> => {
  try {
    return await promise
  } catch (error: any) {
    if (error?.statusCode === 401 || error?.response?.status === 401) {
      localStorage.removeItem('auth_token')
      localStorage.removeItem('username')
      window.location.href = '/login'
    }
    throw error
  }
}

export const dataProvider = {
  ...baseProvider,
  getList: async ({ resource, pagination, filters, sorters, meta }) => {
    const response = await withAuthRedirect(baseProvider.getList({
      resource,
      pagination,
      filters,
      sorters,
      meta: { ...meta, headers: authHeaders() },
    }))
    return response
  },
  getOne: async ({ resource, id, meta }) => {
    return withAuthRedirect(baseProvider.getOne({
      resource,
      id,
      meta: { ...meta, headers: authHeaders() },
    }))
  },
  create: async ({ resource, variables, meta }) => {
    return withAuthRedirect(baseProvider.create({
      resource,
      variables,
      meta: { ...meta, headers: authHeaders() },
    }))
  },
  update: async ({ resource, id, variables, meta }) => {
    return withAuthRedirect(baseProvider.update({
      resource,
      id,
      variables,
      meta: { ...meta, headers: authHeaders() },
    }))
  },
  deleteOne: async ({ resource, id, meta }) => {
    return withAuthRedirect(baseProvider.deleteOne({
      resource,
      id,
      meta: { ...meta, headers: authHeaders() },
    }))
  },
}
