import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  deleteResult,
  deleteSet,
  deleteWorkout,
  finishWorkout,
  getWeek,
  getWorkout,
  listWorkouts,
  logResult,
  logSet,
  startWorkout,
} from '../api'
import { nextWeek, startOf, toIso, weekStart } from '../dates'
import type { Week } from '../types'

// The tracker's data: the week shown, and the api functions for workouts, their sets and the
// scores of their WOD blocks.
export const useWorkoutsStore = defineStore('workouts', () => {
  const week = ref<Week | null>(null)
  const loading = ref(false)

  let request = 0

  // Loads the local week (Monday to Sunday) starting on monday, the current one by default, as
  // one of the active trainings plans it (the first without one). Only the latest request is
  // kept, so stepping quickly through the weeks can't show an older answer last.
  async function loadWeek(trainingId?: string, monday = toIso(weekStart())) {
    const id = ++request
    loading.value = true
    try {
      const start = startOf(monday)
      const result = await getWeek(monday, start, nextWeek(start), trainingId)
      if (id === request) week.value = result
    } finally {
      if (id === request) loading.value = false
    }
  }

  return {
    week,
    loading,
    loadWeek,
    listWorkouts,
    getWorkout,
    startWorkout,
    logSet,
    deleteSet,
    logResult,
    deleteResult,
    finishWorkout,
    deleteWorkout,
  }
})
