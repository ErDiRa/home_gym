import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  connectWhoop,
  disconnectWhoop,
  getWhoopStatus,
  getWorkoutWhoop,
  linkWhoop,
  listWhoopCandidates,
  unlinkWhoop,
} from '../api'
import type { WhoopStatus } from '../types'

// The WHOOP connection and the api functions for linking WHOOP activities to workouts.
export const useWhoopStore = defineStore('whoop', () => {
  const status = ref<WhoopStatus | null>(null)

  async function loadStatus() {
    status.value = await getWhoopStatus()
  }

  return {
    status,
    loadStatus,
    connectWhoop,
    disconnectWhoop,
    listWhoopCandidates,
    getWorkoutWhoop,
    linkWhoop,
    unlinkWhoop,
  }
})
