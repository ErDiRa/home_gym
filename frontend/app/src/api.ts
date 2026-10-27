import createClient from 'openapi-fetch'
import { toast } from 'vue-sonner'
import type { paths } from './openapi'
import { state } from './session'
import type {
  CycleImportInput,
  CycleInput,
  ExerciseInput,
  ExerciseMaxInput,
  ResultInput,
  SetInput,
  TrainingInput,
  TrainingSessionInput,
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

// The API client, typed by the spec (backend/api/openapi.yaml): paths, parameters, bodies and
// responses are checked against it. Same-origin (Vite proxy / reverse proxy), so the auth cookie
// is sent automatically.
const client = createClient<paths>({ baseUrl: '/api' })

client.use({
  onResponse({ request, response }) {
    // The cookie was rejected on a request that needed it: the session is gone.
    if (response.status === 401 && state.user && !new URL(request.url).pathname.endsWith('/auth/login')) {
      state.user = null
      toast.error('Your session has expired. Please sign in again.')
    }
  },
})

// The data of a successful response; a failed one throws ApiError. A non-JSON error body (e.g. a
// proxy's HTML error page) falls back to the status text.
async function unwrap<T>(request: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const { data, error, response } = await request
  if (!response.ok) {
    const message = (error as { error?: string } | undefined)?.error
    throw new ApiError(response.status, typeof message === 'string' ? message : response.statusText)
  }
  return data as T
}

const byId = (id: string) => ({ params: { path: { id } } })

export const listExercises = () => unwrap(client.GET('/exercises'))
export const getExercise = (id: string) => unwrap(client.GET('/exercises/{id}', byId(id)))
export const listJoints = () => unwrap(client.GET('/joints'))
export const listFocuses = () => unwrap(client.GET('/focuses'))
export const listMuscleGroups = () => unwrap(client.GET('/muscle-groups'))
export const createFocus = (name: string, alias: string) => unwrap(client.POST('/focuses', { body: { name, alias } }))
export const createExercise = (input: ExerciseInput) => unwrap(client.POST('/exercises', { body: input }))
export const updateExercise = (id: string, input: ExerciseInput) =>
  unwrap(client.PUT('/exercises/{id}', { ...byId(id), body: input }))
// How the user rated their exercises, for the rating filter in the exercise picker.
export const listExerciseRatings = () => unwrap(client.GET('/exercise-ratings'))
export const rateExercise = (id: string, rating: number, notes: string) =>
  unwrap(client.POST('/exercises/{id}/feedback', { ...byId(id), body: { rating, notes } }))
// The user's 1RM history of all their exercises, newest first per exercise.
export const listExerciseMaxes = () => unwrap(client.GET('/exercise-maxes'))
export const addExerciseMax = (exerciseId: string, input: ExerciseMaxInput) =>
  unwrap(client.POST('/exercises/{id}/maxes', { ...byId(exerciseId), body: input }))
export const deleteExerciseMax = (id: string) => unwrap(client.DELETE('/exercise-maxes/{id}', byId(id)))

export const listBlockKinds = () => unwrap(client.GET('/block-kinds'))
export const listRepUnits = () => unwrap(client.GET('/rep-units'))
export const createRepUnit = (name: string) => unwrap(client.POST('/rep-units', { body: { name } }))
export const listLoadUnits = () => unwrap(client.GET('/load-units'))
export const createLoadUnit = (name: string) => unwrap(client.POST('/load-units', { body: { name } }))

export const listTrainings = () => unwrap(client.GET('/trainings'))
export const getTraining = (id: string) => unwrap(client.GET('/trainings/{id}', byId(id)))
export const createTraining = (input: TrainingInput) => unwrap(client.POST('/trainings', { body: input }))
export const updateTraining = (id: string, input: TrainingInput) =>
  unwrap(client.PUT('/trainings/{id}', { ...byId(id), body: input }))
export const deleteTraining = (id: string) => unwrap(client.DELETE('/trainings/{id}', byId(id)))
// Copies a training (inactive) as the next phase.
export const duplicateTraining = (id: string) => unwrap(client.POST('/trainings/{id}/duplicate', byId(id)))

export const listCycles = () => unwrap(client.GET('/cycles'))
export const getCycle = (id: string) => unwrap(client.GET('/cycles/{id}', byId(id)))
export const createCycle = (input: CycleInput) => unwrap(client.POST('/cycles', { body: input }))
export const updateCycle = (id: string, input: CycleInput) => unwrap(client.PUT('/cycles/{id}', { ...byId(id), body: input }))
// Its trainings are deleted too with withTrainings, else they stay and just leave the cycle.
export const deleteCycle = (id: string, withTrainings: boolean) =>
  unwrap(client.DELETE('/cycles/{id}', { params: { path: { id }, query: { trainings: withTrainings ? 'delete' : 'keep' } } }))

// Creates a cycle with all its trainings, or nothing if any part is invalid.
export const importCycle = (input: CycleImportInput) => unwrap(client.POST('/cycles/import', { body: input }))

// The session library: templates that trainings copy.
export const listSessions = () => unwrap(client.GET('/sessions'))
export const getSession = (id: string) => unwrap(client.GET('/sessions/{id}', byId(id)))
export const createSession = (input: TrainingSessionInput) => unwrap(client.POST('/sessions', { body: input }))
export const updateSession = (id: string, input: TrainingSessionInput) =>
  unwrap(client.PUT('/sessions/{id}', { ...byId(id), body: input }))
export const deleteSession = (id: string) => unwrap(client.DELETE('/sessions/{id}', byId(id)))

// A period [from, to) as query parameters; instants, so the server needs no time zone.
const period = (from: Date, to: Date) => ({ from: from.toISOString(), to: to.toISOString() })

// The week starting on the local Monday from (a YYYY-MM-DD date plus the instants), planned by
// one of the active trainings (the first without one).
export const getWeek = (monday: string, from: Date, to: Date, trainingId?: string) =>
  unwrap(client.GET('/week', { params: { query: { date: monday, ...period(from, to), training: trainingId } } }))
// Workouts finished in [from, to), newest first.
export const listWorkouts = (from: Date, to: Date) => unwrap(client.GET('/workouts', { params: { query: period(from, to) } }))
export const getWorkout = (id: string) => unwrap(client.GET('/workouts/{id}', byId(id)))
// A slot started before (and not finished) returns that workout instead of a new one.
export const startWorkout = (weekSessionId: string) =>
  unwrap(client.POST('/workouts', { body: { week_session_id: weekSessionId } }))
// Logs a set, or corrects it when the same block, position and set number was logged before.
export const logSet = (workoutId: string, set: SetInput) =>
  unwrap(client.PUT('/workouts/{id}/sets', { ...byId(workoutId), body: set }))
export const deleteSet = (workoutId: string, setId: string) =>
  unwrap(client.DELETE('/workouts/{id}/sets/{setID}', { params: { path: { id: workoutId, setID: setId } } }))
// Logs the score of a WOD block, or corrects it when the block's score was logged before.
export const logResult = (workoutId: string, result: ResultInput) =>
  unwrap(client.PUT('/workouts/{id}/results', { ...byId(workoutId), body: result }))
export const deleteResult = (workoutId: string, resultId: string) =>
  unwrap(client.DELETE('/workouts/{id}/results/{resultID}', { params: { path: { id: workoutId, resultID: resultId } } }))
// completedAt (RFC 3339) dates the workout to another day than today; omitted keeps its date.
export const finishWorkout = (id: string, notes: string, completedAt?: string) =>
  unwrap(client.POST('/workouts/{id}/finish', { ...byId(id), body: { notes, completed_at: completedAt } }))
export const deleteWorkout = (id: string) => unwrap(client.DELETE('/workouts/{id}', byId(id)))

// WHOOP: connecting is a full-page visit (WHOOP's login), which returns to /track/settings.
export const getWhoopStatus = () => unwrap(client.GET('/whoop'))
export const connectWhoop = () => window.location.assign('/api/whoop/connect')
export const disconnectWhoop = () => unwrap(client.DELETE('/whoop'))
// The WHOOP activities in [from, to), the workout's local day.
export const listWhoopCandidates = (workoutId: string, from: Date, to: Date) =>
  unwrap(client.GET('/workouts/{id}/whoop/candidates', { params: { path: { id: workoutId }, query: period(from, to) } }))
// The WHOOP data linked to the workout, or null.
export const getWorkoutWhoop = (workoutId: string) => unwrap(client.GET('/workouts/{id}/whoop', byId(workoutId)))
export const linkWhoop = (workoutId: string, whoopWorkoutId: string) =>
  unwrap(client.PUT('/workouts/{id}/whoop', { ...byId(workoutId), body: { whoop_workout_id: whoopWorkoutId } }))
export const unlinkWhoop = (workoutId: string) => unwrap(client.DELETE('/workouts/{id}/whoop', byId(workoutId)))

export async function login(email: string, password: string, remember: boolean) {
  state.user = await unwrap(client.POST('/auth/login', { body: { email, password, remember } }))
}

export async function logout() {
  // Sign out locally even if the request fails, so the UI never stays half-logged-in.
  try {
    await unwrap(client.POST('/auth/logout'))
  } finally {
    state.user = null
  }
}

// Asks the server whether the cookie is still valid. Runs once, before the first navigation.
export async function restoreSession() {
  if (state.restored) return
  try {
    state.user = await unwrap(client.GET('/auth/me'))
  } catch {
    state.user = null
  } finally {
    state.restored = true
  }
}
