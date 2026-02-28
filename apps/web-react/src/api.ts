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
