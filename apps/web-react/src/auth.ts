const TOKEN_KEY = 'mf_access_token'

export function getAccessToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setAccessToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearAccessToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export function authHeaders(): HeadersInit {
  const token = getAccessToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}
