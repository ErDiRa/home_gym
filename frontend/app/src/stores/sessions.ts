import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createSession, deleteSession, getSession, listSessions, updateSession } from '../api'
import type { LibrarySession } from '../types'

// The session library: templates that trainings copy.
export const useSessionsStore = defineStore('sessions', () => {
  const sessions = ref<LibrarySession[]>([])
  const loading = ref(false)

  async function loadSessions() {
    loading.value = true
    try {
      sessions.value = await listSessions()
    } finally {
      loading.value = false
    }
  }

  return { sessions, loading, loadSessions, getSession, createSession, updateSession, deleteSession }
})
