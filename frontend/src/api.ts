import { toast } from 'vue-sonner'
import { state, type User } from './session'
import type {
  BlockKind,
  Client,
  ClientDetail,
  ClientInput,
  Exercise,
  ExerciseDetail,
  ExerciseInput,
  ExerciseRating,
  Focus,
  NewClientInput,
  Option,
  Training,
  TrainingDetail,
  TrainingInput,
  Unit,
} from './types'

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
  delete: (path: string) => request<null>(path, { method: 'DELETE' }),
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

// Trainings and clients are admin-only.
export const listBlockKinds = () => api.get<BlockKind[]>('/admin/block-kinds')
export const listRepUnits = () => api.get<Unit[]>('/admin/rep-units')
export const createRepUnit = (name: string) => api.post<Unit>('/admin/rep-units', { name })
export const listLoadUnits = () => api.get<Unit[]>('/admin/load-units')
export const createLoadUnit = (name: string) => api.post<Unit>('/admin/load-units', { name })
export const listTrainings = () => api.get<Training[]>('/admin/trainings')
export const getTraining = (id: string) => api.get<TrainingDetail>(`/admin/trainings/${id}`)
export const createTraining = (input: TrainingInput) => api.post<Training>('/admin/trainings', input)
export const updateTraining = (id: string, input: TrainingInput) => api.put<Training>(`/admin/trainings/${id}`, input)
export const deleteTraining = (id: string) => api.delete(`/admin/trainings/${id}`)
// Copies a training (inactive) for a client, the same or another one.
export const duplicateTraining = (id: string, clientId: string) =>
  api.post<Training>(`/admin/trainings/${id}/duplicate`, { client_id: clientId })

export const listClients = () => api.get<Client[]>('/admin/clients')
export const getClient = (id: string) => api.get<ClientDetail>(`/admin/clients/${id}`)
export const createClient = (input: NewClientInput) => api.post<Client>('/admin/clients', input)
export const updateClient = (id: string, input: ClientInput) => api.put<Client>(`/admin/clients/${id}`, input)
export const deleteClient = (id: string) => api.delete(`/admin/clients/${id}`)
// How a client rated exercises, for the rating filter in the exercise picker.
export const listClientExerciseRatings = (id: string) =>
  api.get<ExerciseRating[]>(`/admin/clients/${id}/exercise-ratings`)

export async function login(email: string, password: string, remember: boolean) {
  state.user = await api.post<User>('/auth/login', { email, password, remember })
}

export async function logout() {
  // Sign out locally even if the request fails, so the UI never stays half-logged-in.
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
