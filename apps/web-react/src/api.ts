import { authHeaders, clearAccessToken } from './auth'

export async function apiFetch(input: string, init?: RequestInit): Promise<Response> {
  const headers: HeadersInit = {
    ...authHeaders(),
    ...(init?.headers ?? {}),
  }

  const resp = await fetch(input, {
    ...init,
    headers,
  })

  if (resp.status === 401) {
    clearAccessToken()
    if (window.location.pathname !== '/login') {
      window.location.assign('/login')
    }
    throw new Error('登录已失效，请重新登录')
  }

  return resp
}

export const api = {
  async get(url: string) {
    const resp = await apiFetch(url)
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
    }
    return resp.json()
  },

  async post(url: string, data?: any) {
    const resp = await apiFetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: data ? JSON.stringify(data) : undefined,
    })
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
    }
    return resp.json()
  },

  async put(url: string, data?: any) {
    const resp = await apiFetch(url, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: data ? JSON.stringify(data) : undefined,
    })
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
    }
    return resp.json()
  },

  async patch(url: string, data?: any) {
    const resp = await apiFetch(url, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: data ? JSON.stringify(data) : undefined,
    })
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
    }
    return resp.json()
  },

  async delete(url: string) {
    const resp = await apiFetch(url, {
      method: 'DELETE',
    })
    if (!resp.ok) {
      throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
    }
    return resp.json()
  },
}
