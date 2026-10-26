import { toast } from 'vue-sonner'
import { state, type User } from './session'
import type {
  BlockKind,
  Cycle,
  CycleImportInput,
  CycleInput,
  CycleSummary,
  ImportedCycle,
  Exercise,
  ExerciseDetail,
  ExerciseInput,
  ExerciseMax,
  ExerciseMaxInput,
  ExerciseRating,
  Focus,
  LibrarySession,
  LibrarySessionDetail,
  LoggedSet,
  Option,
  ResultInput,
  SetInput,
  Training,
  TrainingDetail,
  TrainingInput,
  TrainingSessionInput,
  Unit,
  Week,
  WhoopCandidate,
  WhoopData,
  WhoopStatus,
  Workout,
  WorkoutResult,
  WorkoutSummary,
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

export const listExercises = () => api.get<Exercise[]>('/exercises')
export const getExercise = (id: string) => api.get<ExerciseDetail>(`/exercises/${id}`)
export const listJoints = () => api.get<Option[]>('/joints')
export const listFocuses = () => api.get<Focus[]>('/focuses')
export const listMuscleGroups = () => api.get<Option[]>('/muscle-groups')
export const createFocus = (name: string, alias: string) => api.post<Focus>('/focuses', { name, alias })
export const createExercise = (input: ExerciseInput) => api.post<Exercise>('/exercises', input)
export const updateExercise = (id: string, input: ExerciseInput) => api.put<Exercise>(`/exercises/${id}`, input)
// How the user rated their exercises, for the rating filter in the exercise picker.
export const listExerciseRatings = () => api.get<ExerciseRating[]>('/exercise-ratings')
export const rateExercise = (id: string, rating: number, notes: string) =>
  api.post<ExerciseRating>(`/exercises/${id}/feedback`, { rating, notes })
// The user's 1RM history of all their exercises, newest first per exercise.
export const listExerciseMaxes = () => api.get<ExerciseMax[]>('/exercise-maxes')
export const addExerciseMax = (exerciseId: string, input: ExerciseMaxInput) =>
  api.post<ExerciseMax>(`/exercises/${exerciseId}/maxes`, input)
export const deleteExerciseMax = (id: string) => api.delete(`/exercise-maxes/${id}`)

export const listBlockKinds = () => api.get<BlockKind[]>('/block-kinds')
export const listRepUnits = () => api.get<Unit[]>('/rep-units')
export const createRepUnit = (name: string) => api.post<Unit>('/rep-units', { name })
export const listLoadUnits = () => api.get<Unit[]>('/load-units')
export const createLoadUnit = (name: string) => api.post<Unit>('/load-units', { name })

export const listTrainings = () => api.get<Training[]>('/trainings')
export const getTraining = (id: string) => api.get<TrainingDetail>(`/trainings/${id}`)
export const createTraining = (input: TrainingInput) => api.post<Training>('/trainings', input)
export const updateTraining = (id: string, input: TrainingInput) => api.put<Training>(`/trainings/${id}`, input)
export const deleteTraining = (id: string) => api.delete(`/trainings/${id}`)
// Copies a training (inactive) as the next phase.
export const duplicateTraining = (id: string) => api.post<Training>(`/trainings/${id}/duplicate`)

export const listCycles = () => api.get<CycleSummary[]>('/cycles')
export const getCycle = (id: string) => api.get<Cycle>(`/cycles/${id}`)
export const createCycle = (input: CycleInput) => api.post<Cycle>('/cycles', input)
export const updateCycle = (id: string, input: CycleInput) => api.put<Cycle>(`/cycles/${id}`, input)
// Its trainings are deleted too with withTrainings, else they stay and just leave the cycle.
export const deleteCycle = (id: string, withTrainings: boolean) =>
  api.delete(`/cycles/${id}?trainings=${withTrainings ? 'delete' : 'keep'}`)

// Creates a cycle with all its trainings, or nothing if any part is invalid.
export const importCycle = (input: CycleImportInput) => api.post<ImportedCycle>('/cycles/import', input)

// The session library: templates that trainings copy.
export const listSessions = () => api.get<LibrarySession[]>('/sessions')
export const getSession = (id: string) => api.get<LibrarySessionDetail>(`/sessions/${id}`)
export const createSession = (input: TrainingSessionInput) => api.post<LibrarySessionDetail>('/sessions', input)
export const updateSession = (id: string, input: TrainingSessionInput) =>
  api.put<LibrarySessionDetail>(`/sessions/${id}`, input)
export const deleteSession = (id: string) => api.delete(`/sessions/${id}`)

// A period [from, to) as query parameters; instants, so the server needs no time zone.
const period = (from: Date, to: Date) =>
  new URLSearchParams({ from: from.toISOString(), to: to.toISOString() }).toString()

// The week starting on the local Monday from (a YYYY-MM-DD date plus the instants), planned by
// one of the active trainings (the first without one).
export const getWeek = (monday: string, from: Date, to: Date, trainingId?: string) =>
  api.get<Week>(`/week?date=${monday}&${period(from, to)}${trainingId ? `&training=${trainingId}` : ''}`)
// Workouts finished in [from, to), newest first.
export const listWorkouts = (from: Date, to: Date) => api.get<WorkoutSummary[]>(`/workouts?${period(from, to)}`)
export const getWorkout = (id: string) => api.get<Workout>(`/workouts/${id}`)
// A slot started before (and not finished) returns that workout instead of a new one.
export const startWorkout = (weekSessionId: string) => api.post<Workout>('/workouts', { week_session_id: weekSessionId })
// Logs a set, or corrects it when the same block, position and set number was logged before.
export const logSet = (workoutId: string, set: SetInput) => api.put<LoggedSet>(`/workouts/${workoutId}/sets`, set)
export const deleteSet = (workoutId: string, setId: string) => api.delete(`/workouts/${workoutId}/sets/${setId}`)
// Logs the score of a WOD block, or corrects it when the block's score was logged before.
export const logResult = (workoutId: string, result: ResultInput) =>
  api.put<WorkoutResult>(`/workouts/${workoutId}/results`, result)
export const deleteResult = (workoutId: string, resultId: string) =>
  api.delete(`/workouts/${workoutId}/results/${resultId}`)
// completedAt (RFC 3339) dates the workout to another day than today; omitted keeps its date.
export const finishWorkout = (id: string, notes: string, completedAt?: string) =>
  api.post<Workout>(`/workouts/${id}/finish`, { notes, completed_at: completedAt })
export const deleteWorkout = (id: string) => api.delete(`/workouts/${id}`)

// WHOOP: connecting is a full-page visit (WHOOP's login), which returns to /track/settings.
export const getWhoopStatus = () => api.get<WhoopStatus>('/whoop')
export const connectWhoop = () => window.location.assign('/api/whoop/connect')
export const disconnectWhoop = () => api.delete('/whoop')
// The WHOOP activities in [from, to), the workout's local day.
export const listWhoopCandidates = (workoutId: string, from: Date, to: Date) =>
  api.get<WhoopCandidate[]>(`/workouts/${workoutId}/whoop/candidates?${period(from, to)}`)
export const getWorkoutWhoop = (workoutId: string) => api.get<WhoopData | null>(`/workouts/${workoutId}/whoop`)
export const linkWhoop = (workoutId: string, whoopWorkoutId: string) =>
  api.put<WhoopData>(`/workouts/${workoutId}/whoop`, { whoop_workout_id: whoopWorkoutId })
export const unlinkWhoop = (workoutId: string) => api.delete(`/workouts/${workoutId}/whoop`)

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

// Asks the server whether the cookie is still valid. Runs once, before the first navigation.
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
