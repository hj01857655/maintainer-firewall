const TOKEN_KEY = 'mf_access_token'
const TENANT_KEY = 'mf_tenant_id'

export function getAccessToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setAccessToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearAccessToken() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(TENANT_KEY)
}

export function getTenantId(): string {
  return localStorage.getItem(TENANT_KEY) || 'default'
}

export function setTenantId(tenantId: string) {
  const normalized = tenantId.trim() || 'default'
  localStorage.setItem(TENANT_KEY, normalized)
}

export function authHeaders(): HeadersInit {
  const token = getAccessToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}
