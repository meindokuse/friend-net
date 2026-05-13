// All requests go through Traefik at port 80.
// Traefik's jwt-auth forwardAuth middleware calls /auth/validate, extracts the
// account_id from the token, and injects X-Account-Id into the upstream request.
// We only need to send Authorization: Bearer <token> — Traefik does the rest.

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://192.168.49.2:32610'
const AUTH_BASE = `${API_BASE}/auth`
const USERS_BASE = `${API_BASE}/users`

export async function apiFetch(
  url: string,
  options: RequestInit = {},
  accessToken?: string,
): Promise<{ data: unknown; status: number; isOk: boolean }> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`

  const res = await fetch(url, { ...options, headers })
  let data: unknown
  const text = await res.text()
  try {
    data = JSON.parse(text)
  } catch {
    data = text
  }
  return { data, status: res.status, isOk: res.ok }
}

// Auth service — routes via Traefik
// Public:  /auth/register, /auth/login, /auth/refresh, /auth/introspect, /auth/google, /healthz
// Private: /auth/logout, /auth/sessions, /auth/link* (jwt-auth middleware)
export const authApi = {
  register: (body: { email: string; password: string; display_name: string }) =>
    apiFetch(`${AUTH_BASE}/register`, { method: 'POST', body: JSON.stringify(body) }),

  login: (body: { email: string; password: string }) =>
    apiFetch(`${AUTH_BASE}/login`, { method: 'POST', body: JSON.stringify(body) }),

  refresh: (refreshToken: string) =>
    apiFetch(`${AUTH_BASE}/refresh`, {
      method: 'POST',
      headers: { 'X-Refresh-Token': refreshToken },
      body: JSON.stringify({}),
    }),

  logout: (accessToken: string, refreshToken: string) =>
    apiFetch(`${AUTH_BASE}/logout`, {
      method: 'POST',
      headers: { 'X-Refresh-Token': refreshToken },
      body: JSON.stringify({}),
    }, accessToken),

  logoutAll: (accessToken: string) =>
    apiFetch(`${AUTH_BASE}/logout-all`, { method: 'POST', body: JSON.stringify({}) }, accessToken),

  sessions: (accessToken: string) =>
    apiFetch(`${AUTH_BASE}/sessions`, { method: 'GET' }, accessToken),

  revokeSession: (accessToken: string, sessionId: string) =>
    apiFetch(`${AUTH_BASE}/sessions/${sessionId}`, { method: 'DELETE' }, accessToken),

  introspect: (accessToken: string) =>
    apiFetch(`${AUTH_BASE}/introspect`, { method: 'POST', body: JSON.stringify({ access_token: accessToken }) }),

  // Private: /auth/link prefix → Traefik applies jwt-auth
  linkedAccounts: (accessToken: string) =>
    apiFetch(`${AUTH_BASE}/linked`, { method: 'GET' }, accessToken),

  unlinkProvider: (accessToken: string, provider: string) =>
    apiFetch(`${AUTH_BASE}/linked/${provider}`, { method: 'DELETE' }, accessToken),
}

// User service — routes via Traefik
// Public (priority 100): POST /users, /users/search, /users/username/*
// Private (priority 1):  everything else under /users — Traefik injects X-Account-Id
export const usersApi = {
  create: (body: { username: string; email?: string; phone?: string; display_name: string; id?: string }) =>
    apiFetch(`${USERS_BASE}/`, { method: 'POST', body: JSON.stringify(body) }),

  // Private — Traefik validates token and injects X-Account-Id for user-service
  getMe: (accessToken: string) =>
    apiFetch(`${USERS_BASE}/me`, { method: 'GET' }, accessToken),

  getById: (id: string) =>
    apiFetch(`${USERS_BASE}/${id}`, { method: 'GET' }),

  getByUsername: (username: string) =>
    apiFetch(`${USERS_BASE}/username/${username}`, { method: 'GET' }),

  getBatch: (ids: string[]) =>
    apiFetch(`${USERS_BASE}/batch`, { method: 'POST', body: JSON.stringify({ ids }) }),

  updateProfile: (accessToken: string, body: { display_name: string; bio?: string; avatar_url?: string; version: number }) =>
    apiFetch(`${USERS_BASE}/me/profile`, { method: 'PATCH', body: JSON.stringify(body) }, accessToken),

  updateSettings: (accessToken: string, body: object) =>
    apiFetch(`${USERS_BASE}/me/settings`, { method: 'PATCH', body: JSON.stringify(body) }, accessToken),

  changeEmail: (accessToken: string, body: { email: string; version: number }) =>
    apiFetch(`${USERS_BASE}/me/email`, { method: 'PATCH', body: JSON.stringify(body) }, accessToken),

  changePhone: (accessToken: string, body: { phone: string; version: number }) =>
    apiFetch(`${USERS_BASE}/me/phone`, { method: 'PATCH', body: JSON.stringify(body) }, accessToken),

  deleteMe: (accessToken: string, version: number) =>
    apiFetch(`${USERS_BASE}/me`, { method: 'DELETE', body: JSON.stringify({ version }) }, accessToken),

  updateLastSeen: (accessToken: string) =>
    apiFetch(`${USERS_BASE}/me/last-seen`, { method: 'POST', body: JSON.stringify({}) }, accessToken),

  search: (query: string, limit = 20, cursor?: string) => {
    const params = new URLSearchParams({ q: query, limit: String(limit) })
    if (cursor) params.set('cursor', cursor)
    return apiFetch(`${USERS_BASE}/search?${params}`, { method: 'GET' })
  },

  list: (limit = 20, cursor?: string) => {
    const params = new URLSearchParams({ limit: String(limit) })
    if (cursor) params.set('cursor', cursor)
    return apiFetch(`${USERS_BASE}/me/list?${params}`, { method: 'GET' }, undefined)
  },
}
