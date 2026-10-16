import { toast } from 'vue-sonner'
import { state, type User } from './session'
import type { Exercise, ExerciseDetail, ExerciseInput, Focus, Option } from './types'

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body) headers.set('Content-Type', 'application/json')

  // Same-origin (Vite proxy / reverse proxy), so the auth cookie is sent automatically.
  const res = await fetch(`/api${path}`, { ...init, headers })
  const body = res.status === 204 ? null : await res.json().catch(() => null)

  if (!res.ok) {
    // The cookie was rejected on a request that needed it: the session is gone.
    if (res.status === 401 && state.user && path !== '/auth/login') {
      state.user = null
      toast.error('Your session has expired. Please sign in again.')
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
}

// Admins use the admin endpoints (which also allow writing); user sessions only have the
// read-only client ones.
const audience = () => (state.user?.is_admin ? '/admin' : '/client')

export const listExercises = () => api.get<Exercise[]>(`${audience()}/exercises`)
export const getExercise = (id: string) => api.get<ExerciseDetail>(`${audience()}/exercises/${id}`)
export const listJoints = () => api.get<Option[]>(`${audience()}/joints`)
export const listFocuses = () => api.get<Focus[]>(`${audience()}/focuses`)
export const listMuscleGroups = () => api.get<Option[]>(`${audience()}/muscle-groups`)

// Writing is admin-only.
export const createFocus = (name: string, alias: string) => api.post<Focus>('/admin/focuses', { name, alias })
export const createExercise = (input: ExerciseInput) => api.post<Exercise>('/admin/exercises', input)
export const updateExercise = (id: string, input: ExerciseInput) =>
  api.put<Exercise>(`/admin/exercises/${id}`, input)

export async function login(email: string, password: string, remember: boolean) {
  state.user = await api.post<User>('/auth/login', { email, password, remember })
}

export async function logout() {
  try {
    await api.post('/auth/logout')
  } finally {
    state.user = null
  }
}

/** Asks the server whether the cookie is still valid. Runs once, before the first navigation. */
export async function restoreSession() {
  if (state.restored) return
  try {
    state.user = await api.get<User>('/auth/me')
  } catch {
    state.user = null
  } finally {
    state.restored = true
  }
}
