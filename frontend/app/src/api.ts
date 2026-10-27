import createClient, { type Middleware } from 'openapi-fetch'
import { toast } from 'vue-sonner'
import type { paths as catalogPaths } from './openapi/catalog'
import type { paths as identityPaths } from './openapi/identity'
import type { paths as planningPaths } from './openapi/planning'
import type { paths as trackingPaths } from './openapi/tracking'
import type { paths as whoopPaths } from './openapi/whoop'
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

// The cookie was rejected on a request that needed it: the session is gone.
const sessionExpired: Middleware = {
  onResponse({ request, response }) {
    if (response.status === 401 && state.user && !new URL(request.url).pathname.endsWith('/auth/login')) {
      state.user = null
      toast.error('Your session has expired. Please sign in again.')
    }
  },
}

// One API client per backend module, typed by that module's spec (backend/modules/<module>/api/http/
// openapi.yaml): paths, parameters, bodies and responses are checked against it. Same-origin (Vite
// proxy / reverse proxy), so the auth cookie is sent automatically.
function newClient<Paths extends object>() {
  const c = createClient<Paths>({ baseUrl: '/api' })
  c.use(sessionExpired)
  return c
}

const identity = newClient<identityPaths>()
const catalog = newClient<catalogPaths>()
const planning = newClient<planningPaths>()
const tracking = newClient<trackingPaths>()
const whoop = newClient<whoopPaths>()

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

export const listExercises = () => unwrap(catalog.GET('/exercises'))
export const getExercise = (id: string) => unwrap(catalog.GET('/exercises/{id}', byId(id)))
export const listJoints = () => unwrap(catalog.GET('/joints'))
export const listFocuses = () => unwrap(catalog.GET('/focuses'))
export const listMuscleGroups = () => unwrap(catalog.GET('/muscle-groups'))
export const createFocus = (name: string, alias: string) => unwrap(catalog.POST('/focuses', { body: { name, alias } }))
export const createExercise = (input: ExerciseInput) => unwrap(catalog.POST('/exercises', { body: input }))
export const updateExercise = (id: string, input: ExerciseInput) =>
  unwrap(catalog.PUT('/exercises/{id}', { ...byId(id), body: input }))
// How the user rated their exercises, for the rating filter in the exercise picker.
export const listExerciseRatings = () => unwrap(catalog.GET('/exercise-ratings'))
export const rateExercise = (id: string, rating: number, notes: string) =>
  unwrap(catalog.POST('/exercises/{id}/feedback', { ...byId(id), body: { rating, notes } }))
// The user's 1RM history of all their exercises, newest first per exercise.
export const listExerciseMaxes = () => unwrap(tracking.GET('/exercise-maxes'))
export const addExerciseMax = (exerciseId: string, input: ExerciseMaxInput) =>
  unwrap(tracking.POST('/exercises/{id}/maxes', { ...byId(exerciseId), body: input }))
export const deleteExerciseMax = (id: string) => unwrap(tracking.DELETE('/exercise-maxes/{id}', byId(id)))

export const listBlockKinds = () => unwrap(catalog.GET('/block-kinds'))
export const listRepUnits = () => unwrap(catalog.GET('/rep-units'))
export const createRepUnit = (name: string) => unwrap(catalog.POST('/rep-units', { body: { name } }))
export const listLoadUnits = () => unwrap(catalog.GET('/load-units'))
export const createLoadUnit = (name: string) => unwrap(catalog.POST('/load-units', { body: { name } }))

export const listTrainings = () => unwrap(planning.GET('/trainings'))
export const getTraining = (id: string) => unwrap(planning.GET('/trainings/{id}', byId(id)))
export const createTraining = (input: TrainingInput) => unwrap(planning.POST('/trainings', { body: input }))
export const updateTraining = (id: string, input: TrainingInput) =>
  unwrap(planning.PUT('/trainings/{id}', { ...byId(id), body: input }))
export const deleteTraining = (id: string) => unwrap(planning.DELETE('/trainings/{id}', byId(id)))
// Copies a training (inactive) as the next phase.
export const duplicateTraining = (id: string) => unwrap(planning.POST('/trainings/{id}/duplicate', byId(id)))

export const listCycles = () => unwrap(planning.GET('/cycles'))
export const getCycle = (id: string) => unwrap(planning.GET('/cycles/{id}', byId(id)))
export const createCycle = (input: CycleInput) => unwrap(planning.POST('/cycles', { body: input }))
export const updateCycle = (id: string, input: CycleInput) => unwrap(planning.PUT('/cycles/{id}', { ...byId(id), body: input }))
// Its trainings are deleted too with withTrainings, else they stay and just leave the cycle.
export const deleteCycle = (id: string, withTrainings: boolean) =>
  unwrap(planning.DELETE('/cycles/{id}', { params: { path: { id }, query: { trainings: withTrainings ? 'delete' : 'keep' } } }))

// Creates a cycle with all its trainings, or nothing if any part is invalid.
export const importCycle = (input: CycleImportInput) => unwrap(planning.POST('/cycles/import', { body: input }))

// The session library: templates that trainings copy.
export const listSessions = () => unwrap(planning.GET('/sessions'))
export const getSession = (id: string) => unwrap(planning.GET('/sessions/{id}', byId(id)))
export const createSession = (input: TrainingSessionInput) => unwrap(planning.POST('/sessions', { body: input }))
export const updateSession = (id: string, input: TrainingSessionInput) =>
  unwrap(planning.PUT('/sessions/{id}', { ...byId(id), body: input }))
export const deleteSession = (id: string) => unwrap(planning.DELETE('/sessions/{id}', byId(id)))

// A period [from, to) as query parameters; instants, so the server needs no time zone.
const period = (from: Date, to: Date) => ({ from: from.toISOString(), to: to.toISOString() })

// The week starting on the local Monday from (a YYYY-MM-DD date plus the instants), planned by
// one of the active trainings (the first without one).
export const getWeek = (monday: string, from: Date, to: Date, trainingId?: string) =>
  unwrap(tracking.GET('/week', { params: { query: { date: monday, ...period(from, to), training: trainingId } } }))
// Workouts finished in [from, to), newest first.
export const listWorkouts = (from: Date, to: Date) => unwrap(tracking.GET('/workouts', { params: { query: period(from, to) } }))
export const getWorkout = (id: string) => unwrap(tracking.GET('/workouts/{id}', byId(id)))
// A slot started before (and not finished) returns that workout instead of a new one.
export const startWorkout = (weekSessionId: string) =>
  unwrap(tracking.POST('/workouts', { body: { week_session_id: weekSessionId } }))
// Logs a set, or corrects it when the same block, position and set number was logged before.
export const logSet = (workoutId: string, set: SetInput) =>
  unwrap(tracking.PUT('/workouts/{id}/sets', { ...byId(workoutId), body: set }))
export const deleteSet = (workoutId: string, setId: string) =>
  unwrap(tracking.DELETE('/workouts/{id}/sets/{setID}', { params: { path: { id: workoutId, setID: setId } } }))
// Logs the score of a WOD block, or corrects it when the block's score was logged before.
export const logResult = (workoutId: string, result: ResultInput) =>
  unwrap(tracking.PUT('/workouts/{id}/results', { ...byId(workoutId), body: result }))
export const deleteResult = (workoutId: string, resultId: string) =>
  unwrap(tracking.DELETE('/workouts/{id}/results/{resultID}', { params: { path: { id: workoutId, resultID: resultId } } }))
// completedAt (RFC 3339) dates the workout to another day than today; omitted keeps its date.
export const finishWorkout = (id: string, notes: string, completedAt?: string) =>
  unwrap(tracking.POST('/workouts/{id}/finish', { ...byId(id), body: { notes, completed_at: completedAt } }))
export const deleteWorkout = (id: string) => unwrap(tracking.DELETE('/workouts/{id}', byId(id)))

// WHOOP: connecting is a full-page visit (WHOOP's login), which returns to /track/settings.
export const getWhoopStatus = () => unwrap(whoop.GET('/whoop'))
export const connectWhoop = () => window.location.assign('/api/whoop/connect')
export const disconnectWhoop = () => unwrap(whoop.DELETE('/whoop'))
// The WHOOP activities in [from, to), the workout's local day.
export const listWhoopCandidates = (workoutId: string, from: Date, to: Date) =>
  unwrap(whoop.GET('/workouts/{id}/whoop/candidates', { params: { path: { id: workoutId }, query: period(from, to) } }))
// The WHOOP data linked to the workout, or null.
export const getWorkoutWhoop = (workoutId: string) => unwrap(whoop.GET('/workouts/{id}/whoop', byId(workoutId)))
export const linkWhoop = (workoutId: string, whoopWorkoutId: string) =>
  unwrap(whoop.PUT('/workouts/{id}/whoop', { ...byId(workoutId), body: { whoop_workout_id: whoopWorkoutId } }))
export const unlinkWhoop = (workoutId: string) => unwrap(whoop.DELETE('/workouts/{id}/whoop', byId(workoutId)))

export async function login(email: string, password: string, remember: boolean) {
  state.user = await unwrap(identity.POST('/auth/login', { body: { email, password, remember } }))
}

export async function logout() {
  // Sign out locally even if the request fails, so the UI never stays half-logged-in.
  try {
    await unwrap(identity.POST('/auth/logout'))
  } finally {
    state.user = null
  }
}

// Asks the server whether the cookie is still valid. Runs once, before the first navigation.
export async function restoreSession() {
  if (state.restored) return
  try {
    state.user = await unwrap(identity.GET('/auth/me'))
  } catch {
    state.user = null
  } finally {
    state.restored = true
  }
}
