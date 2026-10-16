import { reactive } from 'vue'

export interface User {
  id: string
  email: string
}

// The JWT lives in an HttpOnly cookie that scripts cannot read, so the only thing
// kept client-side is the user the server reported, in memory.
export const state = reactive<{ user: User | null; restored: boolean }>({
  user: null,
  restored: false,
})

export function isAuthenticated(): boolean {
  return state.user !== null
}
