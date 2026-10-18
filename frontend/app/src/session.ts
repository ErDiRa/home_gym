import { reactive } from 'vue'

// The signed-in user, as returned by /api/auth/login and /api/auth/me.
export interface User {
  id: string
  email: string
}

// The JWT lives in an HttpOnly cookie that scripts cannot read, so the only thing
// kept client-side is the user the server reported, in memory.
// restored: whether /api/auth/me has been asked yet (see restoreSession).
type SessionState = { user: User | null; restored: boolean }

// In dev, Vite may re-evaluate this module on a hot update while other modules keep the old
// instance, which would reset the user to null. Keeping the state in import.meta.hot.data
// makes every instance share the same object, so editing code doesn't log you out.
const initial = reactive<SessionState>({ user: null, restored: false })
export const state: SessionState = import.meta.hot
  ? (import.meta.hot.data.state ??= initial)
  : initial

export function isAuthenticated(): boolean {
  return state.user !== null
}
