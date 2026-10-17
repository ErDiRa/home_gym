import { state, type User } from './session'
import type { Training } from './types'

// A failed request: the HTTP status plus the server's {"error": ...} message (or the status text).
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
  }
}

// Sends a request to /api and returns the decoded JSON, throwing ApiError on a non-2xx status.
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body) headers.set('Content-Type', 'application/json')

  // Same-origin (Vite proxy / reverse proxy), so the auth cookie is sent automatically.
  const res = await fetch(`/api${path}`, { ...init, headers })
  // 204 has no body; a non-JSON body (e.g. a proxy's HTML error page) becomes null.
  const body = res.status === 204 ? null : await res.json().catch(() => null)

  if (!res.ok) {
    // The cookie was rejected on a request that needed it: the session is gone
    // (the router then sends the user to the login page).
    if (res.status === 401 && state.user && path !== '/client/auth/login') {
      state.user = null
    }
    throw new ApiError(res.status, body?.error ?? res.statusText)
  }
  return body as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'POST', body: data === undefined ? undefined : JSON.stringify(data) }),
  put: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'PUT', body: data === undefined ? undefined : JSON.stringify(data) }),
  delete: (path: string) => request<null>(path, { method: 'DELETE' }),
}

// The client login only accepts user accounts; an admin's credentials fail like wrong ones.
export async function login(email: string, password: string, remember: boolean) {
  state.user = await api.post<User>('/client/auth/login', { email, password, remember })
}

export async function logout() {
  // Sign out locally even if the request fails, so the UI never stays half-logged-in.
  try {
    await api.post('/client/auth/logout')
  } finally {
    state.user = null
  }
}

// Asks the server once who the cookie belongs to; an admin session is rejected, so it counts as
// signed out here.
export async function restoreSession() {
  if (state.restored) return
  try {
    state.user = await api.get<User>('/client/auth/me')
  } catch {
    state.user = null
  } finally {
    state.restored = true
  }
}

// The trainings the signed-in client follows (their active one), with the whole tree.
export const listTrainings = () => api.get<Training[]>('/client/trainings')
